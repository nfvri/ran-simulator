package bandwidth

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-api/go/onos/ransim/metrics"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	log "github.com/sirupsen/logrus"
)

type CQIStats struct {
	NumUEs     int
	UsedPRBsDL int
	UsedPRBsUL int
}

func InitBWPs(sCell *model.Cell, statsPerCQI map[int]CQIStats, availPRBsDL, availPRBsUL int, servedUEs []*model.UE) error {

	// Existing BWPs from topology
	existingCellBwps := map[uint64]*model.Bwp{}
	if len(sCell.Bwps) != 0 {
		for index := range sCell.Bwps {
			bwp := *sCell.Bwps[index]
			existingCellBwps[bwp.ID] = &bwp
		}
		return nil
	}

	AllocateBW(sCell, statsPerCQI, availPRBsDL, availPRBsUL, servedUEs)

	if len(sCell.Bwps) == 0 {
		err := fmt.Errorf("failed to initialize BWPs for simulation")
		log.Error(err)
		return err
	}

	return nil

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

	if enoughBW(tCell, requestedBwps) {
		bwpId := len(tCell.Bwps)
		for index := range requestedBwps {
			bwp := requestedBwps[index]
			bwp.ID = uint64(bwpId)
			ue.Cell.BwpRefs = append(ue.Cell.BwpRefs, bwp)
			tCell.Bwps[bwp.ID] = bwp
			bwpId++
		}
		return
	}

	currAlloc := BwAllocationOf(servedUEs)
	ue.Cell.BwpRefs = requestedBwps
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
			Cell:             tCell,
			ServedUEs:        servedUEs,
			PrevBwAllocation: currAlloc,
			ReqBwAllocation:  reqAlloc,
		}
		pf.apply()
	}
}

func AllocateBW(cell *model.Cell, statsPerCQI map[int]CQIStats, availPRBsDL, availPRBsUL int, servedUEs []*model.UE) {
	// Infer BWP allocation from cell prb measurements
	// pick used prbs if found else resort to total available

	// allocate using selected scheme
	switch cell.ResourceAllocScheme {
	case PROPORTIONAL_FAIR:
	default:
		pf := ProportionalFair{
			Cell:        cell,
			ServedUEs:   servedUEs,
			StatsPerCQI: statsPerCQI,
			AvailPRBsDL: availPRBsDL,
			AvailPRBsUL: availPRBsUL,
		}
		pf.apply()
	}

}

func enoughBW(tCell *model.Cell, requestedBwps []*model.Bwp) bool {
	//TODO: check if UL+DL is sufficient istead of individual checks
	requestedBWDLUe, requestedBWULUe := 0, 0
	for index := range requestedBwps {
		bwp := requestedBwps[index]
		if bwp.Downlink {
			requestedBWDLUe += bwp.Scs * 12 * bwp.NumberOfRBs
		} else {
			requestedBWULUe += bwp.Scs * 12 * bwp.NumberOfRBs
		}
	}
	usedBWDLCell, usedBWULCell := usedBWCell(tCell)

	sufficientBWDL := tCell.Channel.BsChannelBwDL-uint32(usedBWDLCell) > uint32(requestedBWDLUe)
	sufficientBWUL := tCell.Channel.BsChannelBwUL-uint32(usedBWULCell) > uint32(requestedBWULUe)

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
	for _, ue := range ues {
		bwAlloc[ue.IMSI] = make([]model.Bwp, 0, len(ue.Cell.BwpRefs))
		for index := range ue.Cell.BwpRefs {
			bwp := *ue.Cell.BwpRefs[index]
			bwAlloc[ue.IMSI] = append(bwAlloc[ue.IMSI], bwp)
		}
	}
	return bwAlloc
}

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
	prbMeasPerCell := map[uint64]map[string]int{}

	for _, metric := range cellMeasurements {
		if _, exists := prbMeasPerCell[metric.EntityID]; !exists {
			prbMeasPerCell[metric.EntityID] = map[string]int{}
		}
		if _, exists := numUEsByCell[metric.EntityID]; !exists {
			numUEsByCell[metric.EntityID] = map[string]int{}
		}

		value, _ := strconv.Atoi(metric.GetValue())

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
				uesPerCQI := numCellUEs / 15
				for cqi := 1; cqi <= 14; cqi++ {
					numUEsPerCQIByCell[cellNCGI][cqi] = uesPerCQI
				}
				numUEsPerCQIByCell[cellNCGI][15] = numCellUEs - 14*uesPerCQI
			}
		} else {
			for metricName, numUes := range numUEsMetrics {
				cqi, err := strconv.Atoi(strings.Split(metricName, ".")[2])
				if err != nil {
					log.Errorf("Error converting CQI level to integer: %v", err)
					continue
				}
				if MatchesPattern(metricName, ACTIVE_UES_DL_PATTERN) {
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

	usedPRBsDLPerCQIByCell := ConvertMetricKeyToCQIKey(cellUsedPRBsDL, numUEsPerCQIByCell, USED_PRBS_DL_METRIC, USED_PRBS_DL_PATTERN)
	usedPRBsULPerCQIByCell := ConvertMetricKeyToCQIKey(cellUsedPRBsUL, numUEsPerCQIByCell, USED_PRBS_UL_METRIC, USED_PRBS_UL_PATTERN)

	return usedPRBsDLPerCQIByCell, usedPRBsULPerCQIByCell
}

func ConvertMetricKeyToCQIKey(cellUsedPRBs map[uint64]map[string]int, numUEsPerCQIByCell map[uint64]map[int]int, cellMetricName, cqiIndexedMetricPattern string) (usedPRBsPerCQIByCell map[uint64]map[int]int) {
	usedPRBsPerCQIByCell = map[uint64]map[int]int{}

	for cellNCGI, usedPRBsMetrics := range cellUsedPRBs {
		usedPRBsPerCQIByCell[cellNCGI] = map[int]int{}
		prbsToAllocate, onlyCellMetricExists := usedPRBsMetrics[cellMetricName]
		// only Cell Metric exists
		if len(usedPRBsMetrics) == 1 && onlyCellMetricExists {
			usedPRBsPerCQIByCell[cellNCGI] = DisagregateCellUsedPRBs(numUEsPerCQIByCell[cellNCGI], prbsToAllocate)
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

// DisagregateCellUsedPRBs only when no CQI Indexed Metrics exist and the Cell Metric exists.
// If CQI Indexed Metrics exist then, use them and ignore Cell Metric
func DisagregateCellUsedPRBs(numUEsPerCQI map[int]int, prbsToAllocate int) (usedPRBsPerCQI map[int]int) {
	usedPRBsPerCQI = map[int]int{}
	sumCQI := 0
	for cqi, numUEs := range numUEsPerCQI {
		sumCQI += numUEs * cqi
	}
	if sumCQI == 0 {
		log.Warnf("sum cqi for cell's ues is 0")
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
		for cqi := 1; cqi <= 15; cqi++ {
			if numUEsPerCQI[cqi] > 0 && remainingPRBs > 0 {
				usedPRBsPerCQI[cqi]++
				remainingPRBs--
			}
		}
	}
	return
}
