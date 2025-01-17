package bandwidth

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/nfvri/onos-api/go/onos/ransim/metrics"
	"github.com/nfvri/onos-api/go/onos/ransim/types"
	"github.com/nfvri/ran-simulator/pkg/model"
	log "github.com/sirupsen/logrus"
)

type FrequencyRange struct {
	ARFCNStart   uint32
	ARFCNEnd     uint32
	FreqStartMHz float64
	StepHz       float64
}

// Frequency ranges for 5G NR as per 3GPP TS 38.104.
var frequencyRanges = []FrequencyRange{
	{0, 599999, 0, 50000},          // Frequency Range 1 (FR1)
	{600000, 2016666, 3000, 15000}, // Frequency Range 2 (FR2)
}

func CalculateFrequencyMHz(arfcn uint32) float64 {
	for _, rangeInfo := range frequencyRanges {
		if arfcn >= rangeInfo.ARFCNStart && arfcn <= rangeInfo.ARFCNEnd {
			offsetARFCN := float64(arfcn - rangeInfo.ARFCNStart)
			frequency := rangeInfo.FreqStartMHz + (offsetARFCN * rangeInfo.StepHz / 1e6)
			return frequency
		}
	}
	return 0
}

func InitBWPs(sCell *model.Cell, numUEs, usedPRBsDL, usedPRBsUL map[int]int, availPRBsDL, availPRBsUL int, servedUEs []*model.UE) {
	if len(sCell.Bwps) == 0 {
		AllocateBW(sCell, numUEs, usedPRBsDL, usedPRBsUL, availPRBsDL, availPRBsUL, servedUEs)

		if len(sCell.Bwps) == 0 {
			log.Errorf("failed to initialize BWPs for cell: %v", sCell.NCGI)
		}
		return
	}

	// Existing BWPs from topology
	existingCellBwps := []*model.Bwp{}
	for index := range sCell.Bwps {
		bwp := *sCell.Bwps[index]
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

func ReleaseBWPs(sCell *model.Cell, ue *model.UE) []*model.Bwp {
	bwps := make([]*model.Bwp, 0, len(ue.Cell.BwpRefs))
	for index := range ue.Cell.BwpRefs {
		bwp := *ue.Cell.BwpRefs[index]
		bwps = append(bwps, &bwp)
		delete(sCell.Bwps, bwp.ID)
	}
	ue.Cell.BwpRefs = []*model.Bwp{}
	return bwps
}

func ReallocateBW(ue *model.UE, requestedBwps []*model.Bwp, tCell *model.Cell, servedUEs []*model.UE) {

	scaledBwps := getScaledBwps(servedUEs, ue.FiveQi, ue.FiveQi, requestedBwps)
	if isEnough, reqBwps := enoughBW(tCell, requestedBwps, scaledBwps); isEnough {
		ue.Cell.BwpRefs = []*model.Bwp{}
		bwpId := len(tCell.Bwps)
		for index := range reqBwps {
			bwp := reqBwps[index]
			bwp.ID = uint64(bwpId)
			ue.Cell.BwpRefs = append(ue.Cell.BwpRefs, bwp)
			tCell.Bwps[bwp.ID] = bwp
			bwpId++
		}
		return
	}

	ue.Cell.BwpRefs = scaledBwps
	// augment allocation with new ue
	servedUEs = append(servedUEs, ue)
	reqAlloc := BwAllocationOf(servedUEs)

	// delete current allocation
	tCell.Bwps = map[uint64]*model.Bwp{}
	for index := range servedUEs {
		servedUE := servedUEs[index]
		servedUE.Cell.BwpRefs = []*model.Bwp{}
	}

	// reallocate using selected scheme
	switch tCell.ResourceAllocScheme {
	case PROPORTIONAL_FAIR:
	default:
		pf := ProportionalFair{
			Cell:            tCell,
			ServedUEs:       servedUEs,
			IsReallocation:  true,
			ReqBwAllocation: reqAlloc,
		}
		pf.apply()
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

func enoughBW(tCell *model.Cell, requestedBwps, scaledBwps []*model.Bwp) (bool, []*model.Bwp) {
	usedBWDLCell, usedBWULCell := usedBWCell(tCell)

	totalBWDL := 0.0
	totalBWUL := 0.0
	for _, carrier := range tCell.Carriers {
		totalBWDL += MHzToHz(float64(carrier.BsChannelBwDL))
		totalBWUL += MHzToHz(float64(carrier.BsChannelBwUL))
	}

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
	if sufficientBWDL && sufficientBWUL {
		return sufficientBWDL && sufficientBWUL, requestedBwps
	}

	scaledBWDLUe, scaledBWULUe := 0, 0
	for index := range scaledBwps {
		bwp := scaledBwps[index]
		if bwp.Downlink {
			scaledBWDLUe += bwp.Scs * 12 * bwp.NumberOfRBs
		} else {
			scaledBWULUe += bwp.Scs * 12 * bwp.NumberOfRBs
		}
	}
	sufficientBWDL = scaledBWDLUe+usedBWDLCell <= availBWDL
	sufficientBWUL = scaledBWULUe+usedBWULCell <= availBWUL

	return sufficientBWDL && sufficientBWUL, scaledBwps
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
	for ueIndex := range ues {
		ue := ues[ueIndex]
		bwAlloc[ue.IMSI] = make([]model.Bwp, 0, len(ue.Cell.BwpRefs))
		for index := range ue.Cell.BwpRefs {
			bwp := *ue.Cell.BwpRefs[index]
			bwAlloc[ue.IMSI] = append(bwAlloc[ue.IMSI], bwp)
		}
	}
	return bwAlloc
}

func getScaledBwps(ues []*model.UE, ueCQI, cqi int, reqBwps []*model.Bwp) []*model.Bwp {
	cqiBWDL := 0
	cqiBWUL := 0
	cqiUEs := 0
	for _, ue := range ues {
		if cqi == ue.FiveQi {
			cqiUEs++
			for _, bwp := range ue.Cell.BwpRefs {
				if bwp.Downlink {
					cqiBWDL += 12 * bwp.Scs * bwp.NumberOfRBs
				} else {
					cqiBWUL += 12 * bwp.Scs * bwp.NumberOfRBs
				}
			}
		}
	}
	if cqiUEs == 0 {
		if cqi-1 == 0 {
			if ueCQI+1 > 15 {
				return reqBwps
			}
			return getScaledBwps(ues, ueCQI, ueCQI+1, reqBwps)
		}
		if cqi > ueCQI {
			if cqi+1 > 15 {
				return reqBwps
			}
			return getScaledBwps(ues, ueCQI, cqi+1, reqBwps)
		}
		return getScaledBwps(ues, ueCQI, cqi-1, reqBwps)
	}

	avgCqiBWDL := cqiBWDL / cqiUEs
	avgCqiBWUL := cqiBWUL / cqiUEs

	reqPRBsDL := 0
	reqBWDL := 0
	reqPRBsUL := 0
	reqBWUL := 0
	reqBWPsDL := []*model.Bwp{}
	reqBWPsUL := []*model.Bwp{}
	for index := range reqBwps {
		bwp := *reqBwps[index]
		if bwp.Downlink {
			reqPRBsDL++
			reqBWDL += 12 * bwp.Scs * bwp.NumberOfRBs
			reqBWPsDL = append(reqBWPsDL, &bwp)
		} else {
			reqPRBsUL++
			reqBWUL += 12 * bwp.Scs * bwp.NumberOfRBs
			reqBWPsUL = append(reqBWPsUL, &bwp)
		}
	}
	scaledBwpsDL := reqBWPsDL
	if reqBWDL > avgCqiBWDL {
		scaledBwpsDL, _ = generateBWPs(avgCqiBWDL, reqPRBsDL, true)
	}
	scaledBwpsUL := reqBWPsUL
	if reqBWUL > avgCqiBWUL {
		scaledBwpsUL, _ = generateBWPs(avgCqiBWUL, reqPRBsUL, false)
	}

	return append(scaledBwpsDL, scaledBwpsUL...)
}

func MHzToHz(MHz float64) float64 {
	return MHz * 1e6
}

func MHzToGHz(MHz float64) float64 {
	return MHz / 1e3
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

		case metric.Key == AVAIL_PRBS_DL_METRIC:
			prbMeasPerCell[metric.EntityID][AVAIL_PRBS_DL_METRIC] = value

		case metric.Key == AVAIL_PRBS_UL_METRIC:
			prbMeasPerCell[metric.EntityID][AVAIL_PRBS_UL_METRIC] = value

		case metric.Key == USED_PRBS_DL_METRIC:
			prbMeasPerCell[metric.EntityID][USED_PRBS_DL_METRIC] = value

		case MatchesPattern(metric.Key, USED_PRBS_DL_PATTERN):
			prbMeasPerCell[metric.EntityID][metric.Key] = value

		case metric.Key == USED_PRBS_UL_METRIC:
			prbMeasPerCell[metric.EntityID][USED_PRBS_UL_METRIC] = value

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

func GetNumUEsPerBeamQS(sCell *model.Cell, numUEsPerCQI map[int]int) map[model.BeamQS]int {
	numUEsPerCQIByBeam := make(map[model.BeamQS]int)

	totalBeams := 0
	for _, carrier := range sCell.Carriers {
		totalBeams += len(carrier.Beams)
	}

	remainingUEsPerCQI := make(map[int]int)
	for cqi, numUEs := range numUEsPerCQI {
		uesPerBeam := numUEs / totalBeams
		remainingUEsPerCQI[cqi] = numUEs % totalBeams

		for carrierIndex, carrier := range sCell.Carriers {
			for beamIndex := range carrier.Beams {
				beamQS := model.BeamQS{
					BeamID: model.BeamID{NCGI: sCell.NCGI, CarrierIndex: carrierIndex, BeamIndex: beamIndex},
					CQI:    cqi,
				}
				numUEsPerCQIByBeam[beamQS] += uesPerBeam
			}
		}
	}

	for cqi, remainingUEs := range remainingUEsPerCQI {
		if remainingUEs == 0 {
			continue
		}
		for carrierIndex, carrier := range sCell.Carriers {
			for beamIndex := range carrier.Beams {
				if remainingUEs == 0 {
					break
				}
				beamQS := model.BeamQS{
					BeamID: model.BeamID{NCGI: sCell.NCGI, CarrierIndex: carrierIndex, BeamIndex: beamIndex},
					CQI:    cqi,
				}
				numUEsPerCQIByBeam[beamQS]++
				remainingUEs--
			}
		}
	}

	return numUEsPerCQIByBeam
}
