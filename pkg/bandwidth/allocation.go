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

func ReallocateUsedPRBs(cellMeasurements []*metrics.Metric, cellReqLoadMetric metrics.Metric, cqiPRBsMap map[uint64]map[int]float64) {
	prbsPerCQI, cqiIndexedPrbsExist := cqiPRBsMap[cellReqLoadMetric.EntityID]
	if cqiIndexedPrbsExist {
		numPRBsToAllocate, err := strconv.ParseFloat(cellReqLoadMetric.Value, 64)
		if err != nil {
			return
		}

		totalPrbs := 0.0
		for _, numPRBs := range prbsPerCQI {
			totalPrbs += numPRBs
		}

		for metricIndex, numPRBs := range prbsPerCQI {
			metricNewValue := float64(int((numPRBs / totalPrbs) * numPRBsToAllocate))
			cellMeasurements[metricIndex].Value = strconv.FormatFloat(metricNewValue, 'f', -1, 64)
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
	cellUEsMap := map[uint64]map[string]int{}
	cellPrbsMap := map[uint64]map[string]int{}

	for _, metric := range cellMeasurements {
		if _, exists := cellPrbsMap[metric.EntityID]; !exists {
			cellPrbsMap[metric.EntityID] = map[string]int{}
		}
		if _, exists := cellUEsMap[metric.EntityID]; !exists {
			cellUEsMap[metric.EntityID] = map[string]int{}
		}

		value, _ := strconv.Atoi(metric.GetValue())

		switch {
		case metric.Key == ACTIVE_UES_METRIC:
			cellUEsMap[metric.EntityID][ACTIVE_UES_METRIC] = value

		case MatchesPattern(metric.Key, ACTIVE_UES_PATTERN):
			cellUEsMap[metric.EntityID][metric.Key] = value

		case metric.Key == TOTAL_PRBS_DL_METRIC:
			cellPrbsMap[metric.EntityID][TOTAL_PRBS_DL_METRIC] = value

		case metric.Key == TOTAL_PRBS_UL_METRIC:
			cellPrbsMap[metric.EntityID][TOTAL_PRBS_UL_METRIC] = value

		case metric.Key == USED_PRBS_DL_METRIC:
			cellPrbsMap[metric.EntityID][USED_PRBS_DL_METRIC] = value

		case MatchesPattern(metric.Key, USED_PRBS_DL_PATTERN):
			cellPrbsMap[metric.EntityID][metric.Key] = value

		case metric.Key == USED_PRBS_UL_METRIC:
			cellPrbsMap[metric.EntityID][USED_PRBS_UL_METRIC] = value

		case MatchesPattern(metric.Key, USED_PRBS_UL_PATTERN):
			cellPrbsMap[metric.EntityID][metric.Key] = value
		}
	}

	return cellUEsMap, cellPrbsMap
}

func DisagregateCellUes(cellUEsMap map[uint64]map[string]int) {
	for cellNCGI, numUEsMetrics := range cellUEsMap {
		if len(numUEsMetrics) == 1 {
			numCellUEs, onlyCellUEsExists := numUEsMetrics[ACTIVE_UES_METRIC]
			if onlyCellUEsExists {
				uesPerCQI := numCellUEs / 15
				for cqi := 1; cqi <= 15; cqi++ {
					metricName := ACTIVE_UES_METRIC + "." + strconv.Itoa(cqi)
					cellUEsMap[cellNCGI][metricName] = uesPerCQI
				}
			}
		}
	}
}

func DisagregateCellUsedPRBs(cellPRBsMap, cellUEsMap map[uint64]map[string]int) {
	cellUsedPRBsDL := map[uint64]map[string]int{}
	cellUsedPRBsUL := map[uint64]map[string]int{}
	for cellNCGI, prbsMetrics := range cellPRBsMap {
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

	for cellNCGI, usedPRBsDLMetrics := range cellUsedPRBsDL {

		if len(usedPRBsDLMetrics) == 1 {
			cellUsedPRBsDL, onlyCellUsedPRBsDLExists := usedPRBsDLMetrics[USED_PRBS_DL_METRIC]
			if onlyCellUsedPRBsDLExists {
				sumCQI, numUEsPerCQI := getCQIInfo(cellNCGI, cellUEsMap)
				for cqi := 1; cqi <= 15; cqi++ {
					metricName := USED_PRBS_DL_METRIC + "." + strconv.Itoa(cqi)
					usedPRBsDlForCQI := ((numUEsPerCQI[cqi] * cqi) / sumCQI) * cellUsedPRBsDL
					if usedPRBsDlForCQI > 0 {
						cellPRBsMap[cellNCGI][metricName] = usedPRBsDlForCQI
					}
				}
			}
		}

	}

	for cellNCGI, usedPRBsULMetrics := range cellUsedPRBsUL {

		if len(usedPRBsULMetrics) == 1 {
			cellUsedPRBsUL, onlyCellUsedPRBsULExists := usedPRBsULMetrics[USED_PRBS_UL_METRIC]
			if onlyCellUsedPRBsULExists {
				sumCQI, numUEsPerCQI := getCQIInfo(cellNCGI, cellUEsMap)
				for cqi := 1; cqi <= 15; cqi++ {
					metricName := USED_PRBS_UL_METRIC + "." + strconv.Itoa(cqi)
					usedPRBsUlForCQI := ((numUEsPerCQI[cqi] * cqi) / sumCQI) * cellUsedPRBsUL
					if usedPRBsUlForCQI > 0 {
						cellPRBsMap[cellNCGI][metricName] = usedPRBsUlForCQI
					}
				}
			}
		}

	}

}

func getCQIInfo(cellNCGI uint64, cellUEsMap map[uint64]map[string]int) (int, map[int]int) {
	sumCQI := 0
	numUEsPerCQI := map[int]int{}
	for metricName, numUEs := range cellUEsMap[cellNCGI] {
		if metricName != ACTIVE_UES_METRIC {
			cqi, err := strconv.Atoi(strings.Split(metricName, ".")[2])
			if err != nil {
				log.Errorf("Error converting CQI level to integer: %v", err)
				continue
			}
			sumCQI += numUEs * cqi
			numUEsPerCQI[cqi] = numUEs
		}
	}
	return sumCQI, numUEsPerCQI
}
