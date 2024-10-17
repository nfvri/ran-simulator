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
	PROPORTIONAL_FAIR    = "PF"
	ROUND_ROBIN          = "RR"
	USED_PRBS_DL_PATTERN = "RRU.PrbUsedDl.([0-9]|1[0-5])"
	USED_PRBS_UL_PATTERN = "RRU.PrbUsedUl.([0-9]|1[0-5])"
	TOTAL_PRBS_DL_METRIC = "RRU.PrbAvailDl"
	TOTAL_PRBS_UL_METRIC = "RRU.PrbAvailUl"
	USED_PRBS_DL_METRIC  = "RRU.PrbUsedDl"
	USED_PRBS_UL_METRIC  = "RRU.PrbUsedUl"
	ACTIVE_UES_METRIC    = "DRB.MeanActiveUeDl."
)

type AllocationStrategy interface {
	apply(cell *model.Cell, ues []*model.UE)
}

// ==========================================================
// PROPORTIONAL FAIR
// ==========================================================
type ProportionalFair struct {
	UeBwMaxDecPerc      float64
	InitialBwAllocation map[types.IMSI][]model.Bwp
	CurrBwAllocation    map[types.IMSI][]model.Bwp
}

// apply applies Proportional Fair scheduling to assign BWPs to UEs for both downlink and uplink
// ensuring total bandwidth limits are respected
func (s *ProportionalFair) apply(cell *model.Cell, ues []*model.UE) {
	totalRBsDL := int(cell.Channel.BsChannelBwDL)
	totalRBsUL := int(cell.Channel.BsChannelBwUL)

	if totalRBsDL == 0 && totalRBsUL == 0 {
		fmt.Println("No bandwidth available for allocation.")
		return
	}

	sort.SliceStable(ues, func(i, j int) bool {
		return ues[i].FiveQi > ues[j].FiveQi
	})

	ueRatesDL, ueRatesUL, totalRateDL, totalRateUL := s.getUeRates(ues)

	allocatedRBsDL := 0
	allocatedRBsUL := 0
ALLOCATION:
	for _, ue := range ues {
		ueBWPercDL := ueRatesDL[ue.IMSI] / totalRateDL
		ueBWPercUL := ueRatesUL[ue.IMSI] / totalRateUL

		log.Infof("UE %v: ueBWPercDL = %.2f, ueBWPercUL = %.2f\n", ue.IMSI, ueBWPercDL, ueBWPercUL)

		remainingRBsDL := totalRBsDL - allocatedRBsDL
		remainingRBsUL := totalRBsUL - allocatedRBsUL
		if remainingRBsDL >= 0 {
			allocatedBWPsDL, err := s.reallocateBWPs(totalRBsDL, ueBWPercDL, ue, true)
			if err != nil {
				log.Warnf("could not allocate downlink bw for ue: %v, %v", ue.IMSI, err)
				break ALLOCATION
			}
			ue.Cell.BwpRefs = append(ue.Cell.BwpRefs, allocatedBWPsDL...)
		}
		if remainingRBsUL >= 0 {
			allocatedBWPsUL, err := s.reallocateBWPs(totalRBsUL, ueBWPercUL, ue, false)
			if err != nil {
				log.Warnf("could not allocate uplink bw for ue: %v, %v", ue.IMSI, err)
				break ALLOCATION
			}
			ue.Cell.BwpRefs = append(ue.Cell.BwpRefs, allocatedBWPsUL...)

		}
		log.Infof("Assigned BWPs to UE %v (Downlink + Uplink): %v\n", ue.IMSI, ue.Cell.BwpRefs)
	}

	log.Infof("Total Downlink RBs Allocated: %d / %d\n", allocatedRBsDL, totalRBsDL)
	log.Infof("Total Uplink RBs Allocated: %d / %d\n", allocatedRBsUL, totalRBsUL)
}

func (s *ProportionalFair) getUeRates(ues []*model.UE) (ueRatesDL, ueRatesUL map[types.IMSI]float64, totalRateDL, totalRateUL float64) {
	ueRatesDL = make(map[types.IMSI]float64)
	ueRatesUL = make(map[types.IMSI]float64)
	for _, ue := range ues {
		for _, bwp := range s.CurrBwAllocation[ue.IMSI] {
			if bwp.Downlink {
				ueRatesDL[ue.IMSI] += float64(bwp.NumberOfRBs)
			} else {
				ueRatesUL[ue.IMSI] += float64(bwp.NumberOfRBs)
			}
		}
		totalRateDL += ueRatesDL[ue.IMSI]
		totalRateUL += ueRatesUL[ue.IMSI]
	}
	return
}

// reallocateBWPs adjusts the BWPs for the UE based on available bandwidth,
// ensuring that no BWP loses more than s.UeBwMaxDecPerc of its previously allocated bandwidth.
func (s *ProportionalFair) reallocateBWPs(totalRBs int, bwPercentage float64, ue *model.UE, downlink bool) ([]*model.Bwp, error) {

	var uePreviousRBs int
	previousBWPs := s.InitialBwAllocation[ue.IMSI]
	for _, bwp := range s.InitialBwAllocation[ue.IMSI] {
		if bwp.Downlink == downlink {
			uePreviousRBs += bwp.NumberOfRBs
		}
	}

	ueNewRBs := int(float64(totalRBs) * (bwPercentage / 100))
	rbsDecreasePerc := float64(uePreviousRBs) - float64(ueNewRBs)/float64(uePreviousRBs)

	if rbsDecreasePerc > s.UeBwMaxDecPerc {
		return nil, fmt.Errorf("reallocation not feasible! max degrade percentage %v violated", s.UeBwMaxDecPerc)
	}

	newBWPs := make([]*model.Bwp, len(previousBWPs))
	ueAllocatedRbs := 0
	for i, bwp := range previousBWPs {
		remaingRbs := float64(ueNewRBs - ueAllocatedRbs)
		rbsToAllocate := int(math.Min(remaingRbs, float64(bwp.NumberOfRBs)))
		if rbsToAllocate > 0 {
			newBWPs[i] = &model.Bwp{
				ID:          bwp.ID,
				Scs:         bwp.Scs,
				NumberOfRBs: rbsToAllocate,
				Downlink:    downlink,
			}
			ueAllocatedRbs += rbsToAllocate
		}
	}
	return newBWPs, nil
}

func ReallocateUsedPRBs(cellMeasurements []*metrics.Metric, cellReqLoadMetric metrics.Metric, cqiPRBsMap map[uint64]map[int]float64, isDownlink bool) (newMetric *metrics.Metric) {
	prbsPerCQI, cqiIndexedPrbsExist := cqiPRBsMap[cellReqLoadMetric.EntityID]
	if cqiIndexedPrbsExist {
		newAllocation := ProportionalFairPrbs(prbsPerCQI, cellReqLoadMetric.Value)
		for metricIndex, metricValue := range newAllocation {
			cellMeasurements[metricIndex].Value = strconv.FormatFloat(metricValue, 'f', -1, 64)
		}

	} else {
		// add the cell UsedPRBs in a single,newly created, CQI-indexed level --RR

		metricValue, err := strconv.ParseFloat(cellReqLoadMetric.Value, 64)
		if err != nil {
			return
		}
		if isDownlink {
			newMetric = &metrics.Metric{
				EntityID: cellReqLoadMetric.EntityID,
				Key:      USED_PRBS_DL_METRIC + ".7",
				Value:    strconv.FormatFloat(metricValue, 'f', -1, 64),
			}
		} else {
			newMetric = &metrics.Metric{
				EntityID: cellReqLoadMetric.EntityID,
				Key:      USED_PRBS_UL_METRIC + ".7",
				Value:    strconv.FormatFloat(metricValue, 'f', -1, 64),
			}
		}
	}

	return
}

func ProportionalFairPrbs(cqiPRBsMap map[int]float64, prbsToAllocate string) map[int]float64 {
	totalPrbs := 0.0

	result := make(map[int]float64, len(cqiPRBsMap))

	numPRBsToAllocate, err := strconv.ParseFloat(prbsToAllocate, 64)
	if err != nil {
		return result
	}

	for _, numPRBs := range cqiPRBsMap {
		totalPrbs += numPRBs
	}
	for cqi, numPRBs := range cqiPRBsMap {
		result[cqi] = float64(int((numPRBs / totalPrbs) * numPRBsToAllocate))
	}

	return result
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

func CreateCellInfoMaps(cellMeasurements []*metrics.Metric) (map[uint64]map[int]int, map[uint64]map[string]int) {
	//cellPrbsMap[NCGI][CQI]
	cellCQIUEsMap := map[uint64]map[int]int{}
	//cellPrbsMap[NCGI][MetricName]
	cellPrbsMap := map[uint64]map[string]int{}
	for _, metric := range cellMeasurements {
		if _, ok := cellPrbsMap[metric.EntityID]; !ok {
			cellPrbsMap[metric.EntityID] = map[string]int{}
		}
		if strings.Contains(metric.Key, ACTIVE_UES_METRIC) {

			cqi, err := strconv.Atoi(metric.Key[len(ACTIVE_UES_METRIC):])
			if err != nil {
				log.Errorf("Error converting CQI level to integer: %v", err)
				continue
			}

			if _, exists := cellCQIUEsMap[metric.EntityID]; !exists {
				cellCQIUEsMap[metric.EntityID] = make(map[int]int)
			}
			numUEs, _ := strconv.Atoi(metric.GetValue())

			// Metrics in the list are ordered chronologically
			// from oldest at the beginning to newest at the end.
			// Keep the latest metric
			cellCQIUEsMap[metric.EntityID][cqi] = numUEs
		}
		if metric.Key == TOTAL_PRBS_DL_METRIC {
			totalPrbsDl, _ := strconv.Atoi(metric.GetValue())
			cellPrbsMap[metric.EntityID][TOTAL_PRBS_DL_METRIC] = totalPrbsDl
		}
		if metric.Key == TOTAL_PRBS_UL_METRIC {
			totalPrbsUl, _ := strconv.Atoi(metric.GetValue())
			cellPrbsMap[metric.EntityID][TOTAL_PRBS_UL_METRIC] = totalPrbsUl
		}
		if MatchesPattern(metric.Key, USED_PRBS_DL_PATTERN) {
			usedPrbsDl, _ := strconv.Atoi(metric.GetValue())
			cellPrbsMap[metric.EntityID][USED_PRBS_DL_METRIC] += usedPrbsDl
		}
		if MatchesPattern(metric.Key, USED_PRBS_UL_PATTERN) {
			usedPrbsUl, _ := strconv.Atoi(metric.GetValue())
			cellPrbsMap[metric.EntityID][USED_PRBS_UL_METRIC] += usedPrbsUl
		}
	}
	return cellCQIUEsMap, cellPrbsMap
}
