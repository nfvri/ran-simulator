package bandwidth

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-api/go/onos/ransim/metrics"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	log "github.com/sirupsen/logrus"
)

const (
	PROPORTIONAL_FAIR = "PF"
	ROUND_ROBIN       = "RR"

	TOTAL_PRBS_DL_METRIC = "RRU.PrbAvailDl"
	TOTAL_PRBS_UL_METRIC = "RRU.PrbAvailUl"

	USED_PRBS_DL_PATTERN = "RRU.PrbUsedDl.([0-9]|1[0-5])"
	USED_PRBS_DL_METRIC  = "RRU.PrbUsedDl"
	USED_PRBS_UL_PATTERN = "RRU.PrbUsedUl.([0-9]|1[0-5])"
	USED_PRBS_UL_METRIC  = "RRU.PrbUsedUl"

	ACTIVE_UES_METRIC  = "DRB.MeanActiveUeDl"
	ACTIVE_UES_PATTERN = "DRB.MeanActiveUeDl.([0-9]|1[0-5])"
)

type AllocationStrategy interface {
	apply(cell *model.Cell, ues []*model.UE)
}

// ==========================================================
// PROPORTIONAL FAIR
// ==========================================================
type ProportionalFair struct {
	UsedPRBsDLPerCQI map[int]int
	UsedPRBsULPerCQI map[int]int
	TotalPRBsDL      int
	TotalPRBsUL      int
	PrevBwAllocation map[types.IMSI][]model.Bwp
	ReqBwAllocation  map[types.IMSI][]model.Bwp
}

// apply applies Proportional Fair scheduling to assign BWPs to UEs for both downlink and uplink
// ensuring total bandwidth limits are respected
func (s *ProportionalFair) apply(cell *model.Cell, ues []*model.UE) {

	existingAllocation := len(s.PrevBwAllocation) > 0
	totalBWDL := int(cell.Channel.BsChannelBwDL) * 10e5
	totalBWUL := int(cell.Channel.BsChannelBwUL) * 10e5

	//TODO:
	if !existingAllocation {
		if len(s.UsedPRBsDLPerCQI) != 0 && len(s.UsedPRBsULPerCQI) != 0 {
			// totalRBsDL = s.UsedPRBsDLPerCQI
			// totalRBsUL = s.UsedPRBsULPerCQI

		}
	}

	// TODO:
	// for usedPRB -> draw from dist with mean=avgSCS
	// cellBW := 15000000.0
	// usedPRBs := 20.0
	// totalPRBs := 50.0
	// ues := map[int]int{
	// 	1:  1,
	// 	5:  2,
	// 	10: 1,
	// 	15: 1,
	// }
	// utilizedBW := cellBW * (usedPRBs / totalPRBs)
	// averageSCS := cellBW / (12 * totalPRBs)
	// sum_cqis := 0
	// totalUes := 0
	// for cqi, ue := range ues {
	// 	sum_cqis += cqi
	// 	totalUes += ue
	// }

	// fmt.Printf("AVG_SCS: %.2f\n", averageSCS/1000)
	// fmt.Printf("utilizedBW: %.2f", utilizedBW/1000)
	if totalBWDL == 0 && totalBWUL == 0 {
		fmt.Println("No bandwidth available for allocation.")
		return
	}

	sort.SliceStable(ues, func(i, j int) bool {
		return ues[i].FiveQi > ues[j].FiveQi
	})

	var ueRatesDL, ueRatesUL map[types.IMSI]float64
	if existingAllocation {
		ueRatesDL, ueRatesUL = s.getUeRates(ues)
	} else {
		ueRatesDL, ueRatesUL = s.getUeRatesBasedOnCQI(ues)
	}

	allocatedBWDL := 0
	allocatedRBsUL := 0
ALLOCATION:
	for _, ue := range ues {
		log.Infof("UE %v: ueBWPercDL = %.2f, ueBWPercUL = %.2f\n", ue.IMSI, ueRatesDL[ue.IMSI], ueRatesUL[ue.IMSI])

		// downlink
		allocatedBWPsDL, err := s.reallocateBWPs(totalBWDL, ueRatesDL[ue.IMSI], ue, true)
		if err != nil {
			log.Warnf("could not allocate downlink bw for ue: %v, %v", ue.IMSI, err)
			break ALLOCATION
		}
		ue.Cell.BwpRefs = append(ue.Cell.BwpRefs, allocatedBWPsDL...)

		// uplink
		allocatedBWPsUL, err := s.reallocateBWPs(totalBWUL, ueRatesUL[ue.IMSI], ue, false)
		if err != nil {
			log.Warnf("could not allocate uplink bw for ue: %v, %v", ue.IMSI, err)
			break ALLOCATION
		}
		ue.Cell.BwpRefs = append(ue.Cell.BwpRefs, allocatedBWPsUL...)

		log.Infof("Assigned BWPs to UE %v (Downlink + Uplink): %v\n", ue.IMSI, ue.Cell.BwpRefs)
	}

	log.Infof("Total Downlink RBs Allocated: %d / %d\n", allocatedBWDL, totalBWDL)
	log.Infof("Total Uplink RBs Allocated: %d / %d\n", allocatedRBsUL, totalBWUL)
}

func (s *ProportionalFair) getUeRates(ues []*model.UE) (ueRatesDL, ueRatesUL map[types.IMSI]float64) {
	ueRatesDL = make(map[types.IMSI]float64)
	ueRatesUL = make(map[types.IMSI]float64)
	cellRequestedBWDL := 0.0
	cellRequestedBWUL := 0.0

	for _, ue := range ues {
		for _, bwp := range s.ReqBwAllocation[ue.IMSI] {
			if bwp.Downlink {
				ueRatesDL[ue.IMSI] += float64(bwp.NumberOfRBs) * float64(bwp.Scs) * 12
			} else {
				ueRatesUL[ue.IMSI] += float64(bwp.NumberOfRBs) * float64(bwp.Scs) * 12
			}
		}
		cellRequestedBWDL += ueRatesDL[ue.IMSI]
		cellRequestedBWUL += ueRatesUL[ue.IMSI]
	}

	for _, ue := range ues {
		ueRatesDL[ue.IMSI] = ueRatesDL[ue.IMSI] / cellRequestedBWDL
		ueRatesDL[ue.IMSI] = ueRatesUL[ue.IMSI] / cellRequestedBWUL
	}

	return
}

func (s *ProportionalFair) getUeRatesBasedOnCQI(ues []*model.UE) (ueRatesDL, ueRatesUL map[types.IMSI]float64) {
	ueRatesDL = make(map[types.IMSI]float64)
	ueRatesUL = make(map[types.IMSI]float64)
	sumCQIs := 0

	for _, ue := range ues {
		sumCQIs += ue.FiveQi
	}

	for _, ue := range ues {
		ueRatesDL[ue.IMSI] = float64(ue.FiveQi / sumCQIs)
		ueRatesUL[ue.IMSI] = float64(ue.FiveQi / sumCQIs)
	}

	return ueRatesDL, ueRatesUL
}

// reallocateBWPs adjusts the BWPs for the UE based on available bandwidth
func (s *ProportionalFair) reallocateBWPs(totalBW int, ueRate float64, ue *model.UE, downlink bool) ([]*model.Bwp, error) {

	ueNewBW := int(float64(totalBW) * ueRate)
	ueAllocatedBW := 0.0

	newBWPs := []*model.Bwp{}
	previousBWPs := s.PrevBwAllocation[ue.IMSI]
	remaingBW := float64(ueNewBW)
	for i, bwp := range previousBWPs {
		remaingBW -= ueAllocatedBW
		remainingPRBs := remaingBW / (12 * float64(bwp.Scs))
		rbsToAllocate := math.Min(remainingPRBs, float64(bwp.NumberOfRBs))

		// TODO: if prbs to allocate < 1 check if SCS can be reduced so as to fit an additional bwp
		if rbsToAllocate > 0 {
			newBWPs[i] = &model.Bwp{
				ID:          bwp.ID,
				Scs:         bwp.Scs,
				NumberOfRBs: int(rbsToAllocate),
				Downlink:    downlink,
			}
			ueAllocatedBW += float64(int(rbsToAllocate) * 12 * bwp.Scs)
		}
	}
	return newBWPs, nil
}

// ReallocateUsedPRBs only when both the Cell Metric and CQI Indexed Metrics exist.
// If Cell Metric doesn't exist then, use the CQI Indexed Metrics and don't ReallocateUsedPRBs
func ReallocateUsedPRBs(cellMeasurements []*metrics.Metric, cellReqLoadMetric metrics.Metric, prbsPerCQI map[int]float64) {

	numPRBsToAllocate, err := strconv.ParseFloat(cellReqLoadMetric.Value, 64)
	if err != nil {
		log.Warnf("failed to convert string metric value to float64")
		return
	}

	totalPrbs := 0.0
	for _, numPRBs := range prbsPerCQI {
		totalPrbs += numPRBs
	}

	if totalPrbs == 0 {
		log.Warnf("cell's total prbs is 0")
		return
	}

	remainingPRBs := int(numPRBsToAllocate)
	for metricIndex, numPRBs := range prbsPerCQI {
		newPRBs := int((numPRBs / totalPrbs) * numPRBsToAllocate)
		cellMeasurements[metricIndex].Value = strconv.FormatFloat(float64(newPRBs), 'f', -1, 64)
		remainingPRBs -= newPRBs
	}

	for remainingPRBs > 0 {
		for metricIndex := range prbsPerCQI {
			if remainingPRBs > 0 {
				measPRBs, err := strconv.ParseFloat(cellMeasurements[metricIndex].Value, 64)
				if err != nil {
					continue
				}
				cellMeasurements[metricIndex].Value = strconv.FormatFloat(float64(measPRBs+1), 'f', -1, 64)
				remainingPRBs--
			}
		}
	}

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
		case metric.Key == ACTIVE_UES_METRIC:
			numUEsByCell[metric.EntityID][ACTIVE_UES_METRIC] = value

		case MatchesPattern(metric.Key, ACTIVE_UES_PATTERN):
			numUEsByCell[metric.EntityID][metric.Key] = value

		case metric.Key == TOTAL_PRBS_DL_METRIC:
			prbMeasPerCell[metric.EntityID][TOTAL_PRBS_DL_METRIC] = value

		case metric.Key == TOTAL_PRBS_UL_METRIC:
			prbMeasPerCell[metric.EntityID][TOTAL_PRBS_UL_METRIC] = value

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

// DisagregateCellUes only when no CQI Indexed Metrics exist and the Cell Metric exists.
// If CQI Indexed Metrics exist then, use them and ignore Cell Metric
func DisagregateCellUes(numUEsByCell map[uint64]map[string]int) map[uint64]map[int]int {

	numUEsPerCQIByCell := map[uint64]map[int]int{}
	for cellNCGI, numUEsMetrics := range numUEsByCell {
		numUEsPerCQIByCell[cellNCGI] = map[int]int{}
		if len(numUEsMetrics) == 1 {
			numCellUEs, onlyCellUEsExists := numUEsMetrics[ACTIVE_UES_METRIC]
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
				if MatchesPattern(metricName, ACTIVE_UES_PATTERN) {
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

	usedPRBsDLPerCQIByCell := DisagregateCellUsedPRBs(cellUsedPRBsDL, numUEsPerCQIByCell, USED_PRBS_DL_METRIC, USED_PRBS_DL_PATTERN)
	usedPRBsULPerCQIByCell := DisagregateCellUsedPRBs(cellUsedPRBsUL, numUEsPerCQIByCell, USED_PRBS_UL_METRIC, USED_PRBS_UL_PATTERN)

	return usedPRBsDLPerCQIByCell, usedPRBsULPerCQIByCell
}

// DisagregateCellUsedPRBs only when no CQI Indexed Metrics exist and the Cell Metric exists.
// If CQI Indexed Metrics exist then, use them and ignore Cell Metric
func DisagregateCellUsedPRBs(cellUsedPRBs map[uint64]map[string]int, numUEsPerCQIByCell map[uint64]map[int]int, cellMetricName, cqiIndexedMetricPattern string) (usedPRBsPerCQIByCell map[uint64]map[int]int) {
	usedPRBsPerCQIByCell = map[uint64]map[int]int{}

	for cellNCGI, usedPRBsMetrics := range cellUsedPRBs {
		usedPRBsPerCQIByCell[cellNCGI] = map[int]int{}
		prbsToAllocate, onlyCellMetricExists := usedPRBsMetrics[cellMetricName]
		// only Cell Metric exists
		if len(usedPRBsMetrics) == 1 && onlyCellMetricExists {
			sumCQI := 0
			for cqi, numUEs := range numUEsPerCQIByCell[cellNCGI] {
				sumCQI += numUEs * cqi
			}
			if sumCQI == 0 {
				log.Warnf("sum cqi for cell's ues is 0")
				return
			}
			remainingPRBs := prbsToAllocate
			for cqi := 1; cqi <= 15; cqi++ {
				usedPRBsDlForCQI := int((float64((numUEsPerCQIByCell[cellNCGI][cqi] * cqi)) / float64(sumCQI)) * float64(prbsToAllocate))
				if usedPRBsDlForCQI > 0 {
					usedPRBsPerCQIByCell[cellNCGI][cqi] = usedPRBsDlForCQI
					remainingPRBs -= usedPRBsDlForCQI
				}
			}
			for remainingPRBs > 0 {
				for cqi := 1; cqi <= 15; cqi++ {
					if numUEsPerCQIByCell[cellNCGI][cqi] > 0 && remainingPRBs > 0 {
						usedPRBsPerCQIByCell[cellNCGI][cqi]++
						remainingPRBs--
					}
				}
			}

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
