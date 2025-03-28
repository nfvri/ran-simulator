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

const NUM_SUBFRAMES = 10

func InitBWPs(pCell *model.Cell, numUEs, usedPRBsDL, usedPRBsUL map[int]int, availPRBsDL, availPRBsUL int, servedUEs []*model.UE) {
	if len(pCell.Bwps) == 0 {
		AllocatePRBs(pCell, numUEs, usedPRBsDL, usedPRBsUL, availPRBsDL, availPRBsUL, servedUEs)

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

	// use DistributeCellUsedPRBsToCQIs for cell bwps disagreggation, as it has the same
	// functionality but name is kept for better readability in the other use cases
	prbsPerCQI := DistributeCellUsedPRBsToCQIs(numUEs, len(existingCellBwps))
	allocatedBWPs := 0
	for cqi, numPRBsToAllocate := range prbsPerCQI {
		if allocatedBWPs+numPRBsToAllocate <= len(existingCellBwps) {
			allocateBWPsToUEs(pCell.NCGI, existingCellBwps[allocatedBWPs:allocatedBWPs+numPRBsToAllocate], servedUEs, cqi)
			allocatedBWPs += numPRBsToAllocate
		}
	}

}

func CurrPRBsUsed(ue *model.UE) (usedPRBsDL, usedPRBsUL int) {
	usedPRBsDL = 0
	usedPRBsUL = 0
	for c := range ue.ServingCells {
		ueServCell := *ue.ServingCells[c]
		for b := range ueServCell.BwpRefs {
			bwp := ueServCell.BwpRefs[b]
			if bwp.Downlink {
				usedPRBsDL += bwp.NumberOfRBs
			} else {
				usedPRBsUL += bwp.NumberOfRBs
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
		log.Infof("\n\n----------->targetCAScheme: %+v\n\n", targetCAScheme)
		// allocate available bandwidth to ue
		allocateRemainingAvailableBW(targetCAScheme.FixedAllocCells, ue, getServedUEs)

		// reallocate total bw to all served ues
		reallocateBandwidth(targetCAScheme.ReallocCells, getServedUEs, ue)
		return
	}

	log.Infof("ue: %v | targetCells: %+v", ue.IMSI, targetCells)
	reallocateBandwidth(targetCells, getServedUEs, ue)
}

func reallocateBandwidth(reallocationCells []*model.Cell, getServedUEs func(ncgi types.NCGI) []*model.UE, ue *model.UE) {
	for c := range reallocationCells {

		targetCell := reallocationCells[c]
		servedUEs := getServedUEs(targetCell.NCGI)
		// augment allocation with new ue
		servedUEs = append(servedUEs, ue)
		reqAlloc := BwAllocationOf(targetCell.NCGI, servedUEs)

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
		pcc := targetCell.Carriers[0]
		arfcn := utils.If(pcc.ArfcnDL > 0, float64(pcc.ArfcnDL), float64(pcc.ArfcnUL))
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

func AllocatePRBs(cell *model.Cell, numUEs, usedPRBsDL, usedPRBsUL map[int]int, availPRBsDL, availPRBsUL int, servedUEs []*model.UE) {
	// allocate using selected scheme
	switch cell.ResourceAllocScheme {
	case PROPORTIONAL_FAIR:
	default:
		pf := ProportionalFair{
			Cell:             cell,
			ServedUEs:        servedUEs,
			NumUEs:           numUEs,
			UsedPRBsDlPerCQI: usedPRBsDL,
			UsedPRBsUlPerCQI: usedPRBsUL,
			AvailPRBsDL:      availPRBsDL,
			AvailPRBsUL:      availPRBsUL,
		}
		pf.apply()
	}

}

func BwAllocationOf(ncgi types.NCGI, ues []*model.UE) map[types.IMSI][]model.Bwp {
	bwAlloc := map[types.IMSI][]model.Bwp{}
	for index := range ues {
		ue := ues[index]
		ueServCell, _ := ue.GetServingCell(ncgi)
		bwAlloc[ue.IMSI] = make([]model.Bwp, 0, len(ueServCell.BwpRefs))
		for index := range ueServCell.BwpRefs {
			bwp := *ueServCell.BwpRefs[index]
			bwAlloc[ue.IMSI] = append(bwAlloc[ue.IMSI], bwp)
		}
	}
	return bwAlloc
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
	// numUEsByCell[NCGI][MetricName]
	numUEsByCell := map[uint64]map[string]int{}
	// prbMeasPerCell[NCGI][MetricName]
	prbMeasPerCell := map[uint64]map[string]int{}

	for _, metric := range cellMeasurements {

		if _, exists := numUEsByCell[metric.EntityID]; !exists {
			numUEsByCell[metric.EntityID] = map[string]int{}
		}
		if _, exists := prbMeasPerCell[metric.EntityID]; !exists {
			prbMeasPerCell[metric.EntityID] = map[string]int{}
		}

		valueFloat, err := strconv.ParseFloat(metric.GetValue(), 64)
		if err != nil {
			log.Errorf("Failed to convert metric value '%v' to float64: %v", metric.GetValue(), err)
		}
		value := int(valueFloat)

		switch {

		// UE Measurements
		case metric.Key == ACTIVE_UES_DL_METRIC:
			numUEsByCell[metric.EntityID][ACTIVE_UES_DL_METRIC] = value

		case MatchesPattern(metric.Key, ACTIVE_UES_DL_PATTERN):
			numUEsByCell[metric.EntityID][metric.Key] = value

		case metric.Key == ACTIVE_UES_UL_METRIC:
			numUEsByCell[metric.EntityID][ACTIVE_UES_UL_METRIC] = value

		case MatchesPattern(metric.Key, ACTIVE_UES_UL_PATTERN):
			numUEsByCell[metric.EntityID][metric.Key] = value

		// PRB Measurements
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
			usedPRBsPerCQIByCell[cellNCGI] = DistributeCellUsedPRBsToCQIs(numUEsPerCQIByCell[cellNCGI], prbsToAllocate)
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

// DistributeCellUsedPRBsToCQIs only when no CQI Indexed Metrics exist and the Cell Metric exists.
// If CQI Indexed Metrics exist then, use them and ignore Cell Metric
func DistributeCellUsedPRBsToCQIs(numUEsPerCQI map[int]int, prbsToAllocate int) (usedPRBsPerCQI map[int]int) {
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
			carIndex := carrierIndex + 1
			for beamIndex := range carrier.Beams {
				bmIndex := beamIndex + 1
				beamQS := model.BeamQS{
					BeamID: model.BeamID{NCGI: sCell.NCGI, CarrierIndex: carIndex, BeamIndex: bmIndex},
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
			carIndex := carrierIndex + 1
			for beamIndex := range carrier.Beams {
				bmIndex := beamIndex + 1
				if remainingUEs == 0 {
					break
				}
				beamQS := model.BeamQS{
					BeamID: model.BeamID{NCGI: sCell.NCGI, CarrierIndex: carIndex, BeamIndex: bmIndex},
					CQI:    cqi,
				}
				numUEsPerCQIByBeam[beamQS]++
				remainingUEs--
			}
		}
	}

	return numUEsPerCQIByBeam
}
