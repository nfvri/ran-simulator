package bandwidth

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/nfvri/onos-api/go/onos/ransim/metrics"
	"github.com/nfvri/onos-api/go/onos/ransim/types"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func InitBWPs(pCell *model.Cell, numUEs, usedPRBsDL, usedPRBsUL map[int]int, availPRBsDL, availPRBsUL int, servedUEs []*model.UE) {
	if len(pCell.Bwps) == 0 {
		AllocateBW(pCell, numUEs, usedPRBsDL, usedPRBsUL, availPRBsDL, availPRBsUL, servedUEs)

		if len(pCell.Bwps) == 0 {
			log.Errorf("failed to initialize BWPs for cell: %v", pCell.NCGI)
		}
		return
	}

	// Existing BWPs from topology
	existingCellBwps := []*model.Bwp{}
	for index := range pCell.Bwps {
		bwp := *pCell.Bwps[index]
		existingCellBwps = append(existingCellBwps, &bwp)
	}

	// use DisaggregateCellUsedPRBs for cell bwps disagregation, as it has the same
	// functionality but name is kept for better readability in the other use cases
	bwpsPerCQI := DisaggregateCellUsedPRBs(numUEs, len(existingCellBwps))
	allocatedBWPs := 0
	for cqi, numBWPsToAllocate := range bwpsPerCQI {
		if allocatedBWPs+numBWPsToAllocate <= len(existingCellBwps) {
			allocateBWPsToUEs(existingCellBwps[allocatedBWPs:allocatedBWPs+numBWPsToAllocate], servedUEs, cqi)
			allocatedBWPs += numBWPsToAllocate
		}
	}

}

func CurrPRBsUsed(ue *model.UE) (UsedPRBsDL, UsedPRBsUL int) {
	for uecIndex := range ue.ServingCells {
		ueServCell := ue.ServingCells[uecIndex]
		for bwpIndex := range ueServCell.BwpRefs {
			bwp := *ueServCell.BwpRefs[bwpIndex]
			if bwp.Downlink {
				UsedPRBsDL += bwp.NumberOfRBs
			} else {
				UsedPRBsUL += bwp.NumberOfRBs
			}
		}
	}
	return
}

func ReleaseBW(servCells []*model.Cell, ue *model.UE) []*model.Bwp {
	releasedBwps := []*model.Bwp{}
	for cIndex := range servCells {
		servCell := servCells[cIndex]
		ueServCell, isServingCell := ue.GetServingCell(servCell.NCGI)
		if isServingCell {
			for bwpIndex := range ueServCell.BwpRefs {
				bwp := *ueServCell.BwpRefs[bwpIndex]
				releasedBwps = append(releasedBwps, &bwp)
				delete(servCell.Bwps, bwp.ID)
			}
			ueServCell.BwpRefs = []*model.Bwp{}
		}
	}
	return releasedBwps
}

func AllocateBandwidth(
	ue *model.UE,
	releasedBwps []*model.Bwp,
	stoppedServingCells []*model.Cell,
	targetCells []*model.Cell,
	targetCAScheme CAScheme,
	getServedUEs func(ncgi types.NCGI) []*model.UE) {

	if targetCAScheme.CanBeImplemented {
		// allocate available bandwidth to ue
		allocateRemainingAvailableBW(targetCAScheme.FixedAllocCells, ue, getServedUEs)

		// reallocate total bw to all served ues
		reallocateBandwidth(targetCAScheme.ReallocCells, getServedUEs, ue)
		return
	}

	reallocateBandwidth(targetCells, getServedUEs, ue)
}

func reallocateBandwidth(reallocationCells []*model.Cell, getServedUEs func(ncgi types.NCGI) []*model.UE, ue *model.UE) {
	for c := range reallocationCells {

		targetCell := reallocationCells[c]
		servedUEs := getServedUEs(targetCell.NCGI)
		// augment allocation with new ue
		servedUEs = append(servedUEs, ue)
		reqAlloc := BwAllocationOf(servedUEs)

		// delete current cell allocation
		targetCell.Bwps = map[uint64]*model.Bwp{}
		for index := range servedUEs {
			servedUE := servedUEs[index]
			servedUEpCell := servedUE.ServingCells[0]
			servedUEpCell.BwpRefs = []*model.Bwp{}
		}

		// reallocate using selected scheme
		switch targetCell.ResourceAllocScheme {
		case PROPORTIONAL_FAIR:
		default:
			pf := ProportionalFair{
				Cell:            targetCell,
				ServedUEs:       servedUEs,
				IsReallocation:  true,
				ReqBwAllocation: reqAlloc,
			}
			pf.apply()
		}
	}
}

func allocateRemainingAvailableBW(fixedAllocCells []*model.Cell, ue *model.UE, getServedUEs func(ncgi types.NCGI) []*model.UE) {

	for c := range fixedAllocCells {
		targetCell := fixedAllocCells[c]
		_, ueTargetServCell := ue.GetNeighborCell(targetCell.NCGI)

		servedUEs := getServedUEs(targetCell.NCGI)
		arfcn := utils.If(targetCell.Channel.ArfcnDL > 0, float64(targetCell.Channel.ArfcnDL), float64(targetCell.Channel.ArfcnUL))
		fr := GetFR(arfcn)
		scs := NrSCSByCQIPerFR[fr][ue.FiveQi]

		availPRBsUL, availPRBsDL, err := GetCellAvailPRBs(targetCell, servedUEs, ue)
		if err != nil {
			continue
		}

		newBwpULId := uint64(len(targetCell.Bwps) + 1)
		newBwpDLId := uint64(len(targetCell.Bwps) + 2)
		newBwpUL := &model.Bwp{
			ID:          newBwpULId,
			NumberOfRBs: availPRBsUL,
			Scs:         scs,
			Downlink:    false,
		}

		newBwpDL := &model.Bwp{
			ID:          newBwpDLId,
			NumberOfRBs: availPRBsDL,
			Scs:         scs,
			Downlink:    true,
		}

		ueTargetServCell.BwpRefs = append(ueTargetServCell.BwpRefs, newBwpUL, newBwpDL)
		targetCell.Bwps[newBwpULId] = newBwpUL
		targetCell.Bwps[newBwpDLId] = newBwpDL
	}
}

func AllocateBW(cell *model.Cell, numUEs, usedPRBsDL, usedPRBsUL map[int]int, availPRBsDL, availPRBsUL int, servedUEs []*model.UE) {
	// Infer BWP allocation from cell prb measurements
	// pick used prbs if found else resort to total available

	// allocate using selected scheme
	switch cell.ResourceAllocScheme {
	case PROPORTIONAL_FAIR:
	default:
		pf := ProportionalFair{
			Cell:        cell,
			ServedUEs:   servedUEs,
			NumUEs:      numUEs,
			UsedPRBsDL:  usedPRBsDL,
			UsedPRBsUL:  usedPRBsUL,
			AvailPRBsDL: availPRBsDL,
			AvailPRBsUL: availPRBsUL,
		}
		pf.apply()
	}

}

func enoughBW(tCell *model.Cell, requestedBwps []*model.Bwp) bool {
	usedBWDLCell, usedBWULCell := usedBWCell(tCell)

	totalBWDL := MHzToHz(float64(tCell.Channel.BsChannelBwDL))
	totalBWUL := MHzToHz(float64(tCell.Channel.BsChannelBwUL))

	availBWDL := int(totalBWDL * DEFAULT_MAX_BW_UTILIZATION)
	availBWUL := int(totalBWUL * DEFAULT_MAX_BW_UTILIZATION)

	requestedBWDLUe, requestedBWULUe := 0, 0
	for index := range requestedBwps {
		bwp := requestedBwps[index]
		if bwp.Downlink {
			requestedBWDLUe += bwp.Scs * 12 * bwp.NumberOfRBs
		} else {
			requestedBWULUe += bwp.Scs * 12 * bwp.NumberOfRBs
		}
	}
	sufficientBWDL := requestedBWDLUe+usedBWDLCell <= availBWDL
	sufficientBWUL := requestedBWULUe+usedBWULCell <= availBWUL

	return sufficientBWDL && sufficientBWUL
}

func enoughBWScaled(tCell *model.Cell, scaledBwps []*model.Bwp) bool {
	usedBWDLCell, usedBWULCell := usedBWCell(tCell)
	totalBWDL := MHzToHz(float64(tCell.Channel.BsChannelBwDL))
	totalBWUL := MHzToHz(float64(tCell.Channel.BsChannelBwUL))

	availBWDL := int(totalBWDL * DEFAULT_MAX_BW_UTILIZATION)
	availBWUL := int(totalBWUL * DEFAULT_MAX_BW_UTILIZATION)
	scaledBWDLUe, scaledBWULUe := 0, 0
	for index := range scaledBwps {
		bwp := scaledBwps[index]
		if bwp.Downlink {
			scaledBWDLUe += bwp.Scs * 12 * bwp.NumberOfRBs
		} else {
			scaledBWULUe += bwp.Scs * 12 * bwp.NumberOfRBs
		}
	}
	sufficientBWDL := scaledBWDLUe+usedBWDLCell <= availBWDL
	sufficientBWUL := scaledBWULUe+usedBWULCell <= availBWUL

	return sufficientBWDL && sufficientBWUL
}

func usedBWCell(cell *model.Cell) (usedBWDLCell, usedBWULCell int) {

	for index := range cell.Bwps {
		bwp := cell.Bwps[index]
		if bwp.Downlink {
			usedBWDLCell += bwp.Scs * 12 * bwp.NumberOfRBs
		} else {
			usedBWULCell += bwp.Scs * 12 * bwp.NumberOfRBs
		}
	}
	return

}

func BwAllocationOf(ues []*model.UE) map[types.IMSI][]model.Bwp {
	bwAlloc := map[types.IMSI][]model.Bwp{}
	for index := range ues {
		ue := ues[index]
		bwAlloc[ue.IMSI] = make([]model.Bwp, 0, len(ue.ServingCells[0].BwpRefs))
		for index := range ue.ServingCells[0].BwpRefs {
			bwp := *ue.ServingCells[0].BwpRefs[index]
			bwAlloc[ue.IMSI] = append(bwAlloc[ue.IMSI], bwp)
		}
	}
	return bwAlloc
}

// TODO: refactor
// func getScaledBwps(ues []*model.UE, ueCQI, cqi int, reqBwps []*model.Bwp) []*model.Bwp {
// 	cqiBWDL := 0
// 	cqiBWUL := 0
// 	cqiNumUEs := 0
// 	for _, ue := range ues {
// 		if cqi == ue.FiveQi {
// 			cqiNumUEs++
// 			for _, bwp := range ue.ServingCells[0].BwpRefs {
// 				if bwp.Downlink {
// 					cqiBWDL += 12 * bwp.Scs * bwp.NumberOfRBs
// 				} else {
// 					cqiBWUL += 12 * bwp.Scs * bwp.NumberOfRBs
// 				}
// 			}
// 		}
// 	}
// 	if cqiNumUEs == 0 {
// 		if cqi-1 == 0 {
// 			if ueCQI+1 > 15 {
// 				return reqBwps
// 			}
// 			return getScaledBwps(ues, ueCQI, ueCQI+1, reqBwps)
// 		}
// 		if cqi > ueCQI {
// 			if cqi+1 > 15 {
// 				return reqBwps
// 			}
// 			return getScaledBwps(ues, ueCQI, cqi+1, reqBwps)
// 		}
// 		return getScaledBwps(ues, ueCQI, cqi-1, reqBwps)
// 	}

// 	avgCqiBWDL := cqiBWDL / cqiNumUEs
// 	avgCqiBWUL := cqiBWUL / cqiNumUEs

// 	reqPRBsDL := 0
// 	reqBWDL := 0
// 	reqPRBsUL := 0
// 	reqBWUL := 0
// 	reqBWPsDL := []*model.Bwp{}
// 	reqBWPsUL := []*model.Bwp{}
// 	for index := range reqBwps {
// 		bwp := *reqBwps[index]
// 		if bwp.Downlink {
// 			reqPRBsDL++
// 			reqBWDL += 12 * bwp.Scs * bwp.NumberOfRBs
// 			reqBWPsDL = append(reqBWPsDL, &bwp)
// 		} else {
// 			reqPRBsUL++
// 			reqBWUL += 12 * bwp.Scs * bwp.NumberOfRBs
// 			reqBWPsUL = append(reqBWPsUL, &bwp)
// 		}
// 	}
// 	scaledBwpsDL := reqBWPsDL
// 	if reqBWDL > avgCqiBWDL {
// 		// TODO: see how to obtain scsOptions
// 		scaledBwpsDL, _ = generateBWPs(avgCqiBWDL, reqPRBsDL, true)
// 	}
// 	scaledBwpsUL := reqBWPsUL
// 	if reqBWUL > avgCqiBWUL {
// 		// TODO: see how to obtain scsOptions
// 		scaledBwpsUL, _ = generateBWPs(avgCqiBWUL, reqPRBsUL, false)
// 	}

// 	return append(scaledBwpsDL, scaledBwpsUL...)
// }

func MHzToHz(MHz float64) float64 {
	return MHz * 1e6
}

func CreateUsedPrbsMaps(cellMeasurements []*metrics.Metric) (map[uint64]map[int]float64, map[uint64]map[int]float64) {
	//cqiPRBsDlMap[NCGI][metricIndex]#PRBs
	cqiPRBsDlMap := map[uint64]map[int]float64{}

	//cqiPRBsUlMap[NCGI][metricIndex]#PRBs
	cqiPRBsUlMap := map[uint64]map[int]float64{}

	for metricIndex, metric := range cellMeasurements {
		if MatchesPattern(metric.Key, USED_PRBS_DL_PATTERN) {
			if _, ok := cqiPRBsDlMap[metric.EntityID]; !ok {
				cqiPRBsDlMap[metric.EntityID] = map[int]float64{}
			}
			value, err := strconv.ParseFloat(metric.Value, 64)
			if err != nil {
				continue
			}
			cqiPRBsDlMap[metric.EntityID][metricIndex] = value
		}
		if MatchesPattern(metric.Key, USED_PRBS_UL_PATTERN) {
			if _, ok := cqiPRBsUlMap[metric.EntityID]; !ok {
				cqiPRBsUlMap[metric.EntityID] = map[int]float64{}
			}
			value, err := strconv.ParseFloat(metric.Value, 64)
			if err != nil {
				continue
			}
			cqiPRBsUlMap[metric.EntityID][metricIndex] = value
		}
	}
	return cqiPRBsDlMap, cqiPRBsUlMap
}

func MatchesPattern(metric, p string) bool {
	r, err := regexp.Compile(p)
	if err != nil {
		return false
	}
	return r.MatchString(metric)
}

func UtilizationInfoByCell(cellMeasurements []*metrics.Metric) (map[uint64]map[string]int, map[uint64]map[string]int) {
	// cellPrbsMap[NCGI][MetricName]
	numUEsByCell := map[uint64]map[string]int{}
	// prbMeasPerCell[NCGI][MetricName]
	prbMeasPerCell := map[uint64]map[string]int{}

	for _, metric := range cellMeasurements {
		if _, exists := prbMeasPerCell[metric.EntityID]; !exists {
			prbMeasPerCell[metric.EntityID] = map[string]int{}
		}
		if _, exists := numUEsByCell[metric.EntityID]; !exists {
			numUEsByCell[metric.EntityID] = map[string]int{}
		}

		valueFloat, err := strconv.ParseFloat(metric.GetValue(), 64)
		if err != nil {
			log.Errorf("Failed to convert metric valye '%v' to float64: %v", metric.GetValue(), err)
		}
		value := int(valueFloat)
		switch {
		case metric.Key == ACTIVE_UES_DL_METRIC:
			numUEsByCell[metric.EntityID][ACTIVE_UES_DL_METRIC] = value

		case MatchesPattern(metric.Key, ACTIVE_UES_DL_PATTERN):
			numUEsByCell[metric.EntityID][metric.Key] = value

		case metric.Key == ACTIVE_UES_UL_METRIC:
			numUEsByCell[metric.EntityID][ACTIVE_UES_UL_METRIC] = value

		case MatchesPattern(metric.Key, ACTIVE_UES_DL_PATTERN):
			numUEsByCell[metric.EntityID][metric.Key] = value

		// TODO: multiply by NUM_SUBFRAMES
		case metric.Key == AVAIL_PRBS_DL_METRIC:
			prbMeasPerCell[metric.EntityID][AVAIL_PRBS_DL_METRIC] = value

		// TODO: multiply by NUM_SUBFRAMES
		case metric.Key == AVAIL_PRBS_UL_METRIC:
			prbMeasPerCell[metric.EntityID][AVAIL_PRBS_UL_METRIC] = value

		// TODO: multiply by NUM_SUBFRAMES
		case metric.Key == USED_PRBS_DL_METRIC:
			prbMeasPerCell[metric.EntityID][USED_PRBS_DL_METRIC] = value

		// TODO: multiply by NUM_SUBFRAMES
		case MatchesPattern(metric.Key, USED_PRBS_DL_PATTERN):
			prbMeasPerCell[metric.EntityID][metric.Key] = value

		// TODO: multiply by NUM_SUBFRAMES
		case metric.Key == USED_PRBS_UL_METRIC:
			prbMeasPerCell[metric.EntityID][USED_PRBS_UL_METRIC] = value

		// TODO: multiply by NUM_SUBFRAMES
		case MatchesPattern(metric.Key, USED_PRBS_UL_PATTERN):
			prbMeasPerCell[metric.EntityID][metric.Key] = value
		}
	}

	return numUEsByCell, prbMeasPerCell
}

// GetNumUEsPerCQIByCell only when no CQI Indexed Metrics exist and the Cell Metric exists.
// If CQI Indexed Metrics exist then, use them and ignore Cell Metric
func GetNumUEsPerCQIByCell(numUEsByCell map[uint64]map[string]int) map[uint64]map[int]int {

	numUEsPerCQIByCell := map[uint64]map[int]int{}
	for cellNCGI, numUEsMetrics := range numUEsByCell {
		numUEsPerCQIByCell[cellNCGI] = map[int]int{}
		if len(numUEsMetrics) == 1 {
			numCellUEs, onlyCellUEsExists := numUEsMetrics[ACTIVE_UES_DL_METRIC]
			if onlyCellUEsExists {
				remainingCellUEs := numCellUEs
				uesPerCQI := numCellUEs / 15
				for cqi := 1; cqi <= 15; cqi++ {
					numUEsPerCQIByCell[cellNCGI][cqi] = uesPerCQI
					remainingCellUEs -= uesPerCQI
				}
				for remainingCellUEs > 0 {
					for cqi := 15; cqi >= 0; cqi-- {
						if remainingCellUEs > 0 {
							numUEsPerCQIByCell[cellNCGI][cqi]++
							remainingCellUEs--
						}
					}
				}
			}
		} else {
			for metricName, numUes := range numUEsMetrics {
				if MatchesPattern(metricName, ACTIVE_UES_DL_PATTERN) {
					cqi, err := strconv.Atoi(strings.Split(metricName, ".")[2])
					if err != nil {
						log.Errorf("Error converting CQI level to integer: %v", err)
						continue
					}
					numUEsPerCQIByCell[cellNCGI][cqi] = numUes
				}
			}
		}
	}
	return numUEsPerCQIByCell
}

func GetUsedPRBsPerCQIByCell(prbMeasPerCell map[uint64]map[string]int, numUEsPerCQIByCell map[uint64]map[int]int) (map[uint64]map[int]int, map[uint64]map[int]int) {
	cellUsedPRBsDL := map[uint64]map[string]int{}
	cellUsedPRBsUL := map[uint64]map[string]int{}
	for cellNCGI, prbsMetrics := range prbMeasPerCell {
		for metricName, numPrbs := range prbsMetrics {
			switch {
			case MatchesPattern(metricName, USED_PRBS_DL_PATTERN) || metricName == USED_PRBS_DL_METRIC:
				if _, exists := cellUsedPRBsDL[cellNCGI]; !exists {
					cellUsedPRBsDL[cellNCGI] = map[string]int{}
				}
				cellUsedPRBsDL[cellNCGI][metricName] = numPrbs

			case MatchesPattern(metricName, USED_PRBS_UL_PATTERN) || metricName == USED_PRBS_UL_METRIC:
				if _, exists := cellUsedPRBsUL[cellNCGI]; !exists {
					cellUsedPRBsUL[cellNCGI] = map[string]int{}
				}
				cellUsedPRBsUL[cellNCGI][metricName] = numPrbs
			}
		}
	}

	usedPRBsDLPerCQIByCell := ConvertMetricKeyToCQIKey(
		cellUsedPRBsDL,
		numUEsPerCQIByCell,
		USED_PRBS_DL_METRIC,
		USED_PRBS_DL_PATTERN,
	)
	usedPRBsULPerCQIByCell := ConvertMetricKeyToCQIKey(
		cellUsedPRBsUL,
		numUEsPerCQIByCell,
		USED_PRBS_UL_METRIC,
		USED_PRBS_UL_PATTERN,
	)

	return usedPRBsDLPerCQIByCell, usedPRBsULPerCQIByCell
}

func ConvertMetricKeyToCQIKey(cellUsedPRBs map[uint64]map[string]int, numUEsPerCQIByCell map[uint64]map[int]int, cellMetricName, cqiIndexedMetricPattern string) (usedPRBsPerCQIByCell map[uint64]map[int]int) {
	usedPRBsPerCQIByCell = map[uint64]map[int]int{}

	for cellNCGI, usedPRBsMetrics := range cellUsedPRBs {
		usedPRBsPerCQIByCell[cellNCGI] = map[int]int{}
		prbsToAllocate, onlyCellMetricExists := usedPRBsMetrics[cellMetricName]
		// only Cell Level Metric exists
		if len(usedPRBsMetrics) == 1 && onlyCellMetricExists {
			usedPRBsPerCQIByCell[cellNCGI] = DisaggregateCellUsedPRBs(numUEsPerCQIByCell[cellNCGI], prbsToAllocate)
		} else {
			for metricName, numPrbs := range usedPRBsMetrics {
				if MatchesPattern(metricName, cqiIndexedMetricPattern) {
					cqi, err := strconv.Atoi(strings.Split(metricName, ".")[2])
					if err != nil {
						log.Errorf("Error converting CQI level to integer: %v", err)
						continue
					}
					usedPRBsPerCQIByCell[cellNCGI][cqi] = numPrbs
				}
			}
		}
	}
	return
}

// DisaggregateCellUsedPRBs only when no CQI Indexed Metrics exist and the Cell Metric exists.
// If CQI Indexed Metrics exist then, use them and ignore Cell Metric
func DisaggregateCellUsedPRBs(numUEsPerCQI map[int]int, prbsToAllocate int) (usedPRBsPerCQI map[int]int) {
	usedPRBsPerCQI = map[int]int{}
	sumCQI := 0
	for cqi, numUEs := range numUEsPerCQI {
		sumCQI += numUEs * cqi
	}
	if sumCQI == 0 {
		log.Warnf("sum cqi for cell's ues is 0")
		return
	}

	if prbsToAllocate == 0 {
		for cqi := 1; cqi <= 15; cqi++ {
			usedPRBsPerCQI[cqi] = 0
		}
		return
	}

	remainingPRBs := prbsToAllocate
	for cqi := 1; cqi <= 15; cqi++ {
		usedPRBsDlForCQI := int((float64((numUEsPerCQI[cqi] * cqi)) / float64(sumCQI)) * float64(prbsToAllocate))
		if usedPRBsDlForCQI > 0 {
			usedPRBsPerCQI[cqi] = usedPRBsDlForCQI
			remainingPRBs -= usedPRBsDlForCQI
		}
	}
	for remainingPRBs > 0 {
		for cqi := 15; cqi >= 0; cqi-- {
			if numUEsPerCQI[cqi] > 0 && remainingPRBs > 0 {
				usedPRBsPerCQI[cqi]++
				remainingPRBs--
			}
		}
	}
	return
}

func CheckBWOverflow(usedPRBsPerCQIByCell map[uint64]map[int]int, prbMeasPerCell map[uint64]map[string]int, cellMetricName string) map[uint64]map[int]int {

	for ncgi, usedPRBsPerCQI := range usedPRBsPerCQIByCell {

		availPRBs := prbMeasPerCell[ncgi][cellMetricName]

		sumUsedPRBs := 0
		for _, usedPRBs := range usedPRBsPerCQI {
			sumUsedPRBs += usedPRBs
		}
		if sumUsedPRBs <= availPRBs {
			continue
		}

		assignedPrbs := 0
		for cqi := range usedPRBsPerCQI {
			usedPRBs := usedPRBsPerCQI[cqi]
			usedPRBsPerCQI[cqi] = int(float64(usedPRBs) / (float64(sumUsedPRBs) / float64(availPRBs)))
			assignedPrbs += usedPRBsPerCQI[cqi]
		}
		if assignedPrbs == availPRBs {
			return usedPRBsPerCQIByCell
		}

		for cqi := range usedPRBsPerCQI {
			usedPRBs := usedPRBsPerCQI[cqi]
			usedPRBsPerCQI[cqi] = usedPRBs + 1
			assignedPrbs++
			if assignedPrbs == availPRBs {
				break
			}
		}

	}
	return usedPRBsPerCQIByCell
}
