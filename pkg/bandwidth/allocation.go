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

	AVAIL_PRBS_DL_METRIC = "RRU.PrbAvailDl"
	AVAIL_PRBS_UL_METRIC = "RRU.PrbAvailUl"

	USED_PRBS_DL_PATTERN = "RRU.PrbUsedDl.([0-9]|1[0-5])"
	USED_PRBS_DL_METRIC  = "RRU.PrbUsedDl"
	USED_PRBS_UL_PATTERN = "RRU.PrbUsedUl.([0-9]|1[0-5])"
	USED_PRBS_UL_METRIC  = "RRU.PrbUsedUl"

	ACTIVE_UES_DL_METRIC  = "DRB.MeanActiveUeDl"
	ACTIVE_UES_DL_PATTERN = "DRB.MeanActiveUeDl.([0-9]|1[0-5])"
	ACTIVE_UES_UL_METRIC  = "DRB.MeanActiveUeUl"
	ACTIVE_UES_UL_PATTERN = "DRB.MeanActiveUeUl.([0-9]|1[0-5])"

	UE_THP_DL_METRIC = "DRB.UEThpDl"
	UE_THP_UL_METRIC = "DRB.UEThpUl"

	DEFAULT_MAX_BW_UTILIZATION = 0.95
)

type AllocationStrategy interface {
	apply(cell *model.Cell, servedUEs []*model.UE)
}

// ==========================================================
// PROPORTIONAL FAIR
// ==========================================================
type ProportionalFair struct {
	StatsPerCQI      map[int]CQIStats
	AvailPRBsDL      int
	AvailPRBsUL      int
	PrevBwAllocation map[types.IMSI][]model.Bwp
	ReqBwAllocation  map[types.IMSI][]model.Bwp
}

// apply applies Proportional Fair scheduling to assign BWPs to UEs for both downlink and uplink
// ensuring total bandwidth limits are respected
func (s *ProportionalFair) apply(cell *model.Cell, servedUEs []*model.UE) {

	existingAllocation := len(s.PrevBwAllocation) > 0
	totalBWDL := ToHz(float64(cell.Channel.BsChannelBwDL))
	totalBWUL := ToHz(float64(cell.Channel.BsChannelBwUL))

	availBWDL := int(totalBWDL * DEFAULT_MAX_BW_UTILIZATION)
	availBWUL := int(totalBWUL * DEFAULT_MAX_BW_UTILIZATION)

	if s.AvailPRBsDL != 0 {
		availBWDL = s.AvailPRBsDL
	}

	if s.AvailPRBsUL != 0 {
		availBWUL = s.AvailPRBsUL
	}

	// Assign bw in descending order of CQI
	// i.e. highest QOS first
	sort.SliceStable(servedUEs, func(i, j int) bool {
		return servedUEs[i].FiveQi > servedUEs[j].FiveQi
	})
	if existingAllocation {
		s.reallocateBW(servedUEs, availBWDL, availBWUL)
	} else {
		s.allocateBW(servedUEs, availBWDL, availBWUL)
	}

	if availBWDL == 0 && availBWUL == 0 {
		fmt.Println("No bandwidth available for allocation.")
		return
	}

}

func (s *ProportionalFair) allocateBW(servedUEs []*model.UE, availBWDL, availBWUL int) {
	scsOptions := []int{15_000, 30_000, 60_000, 120_000}

	for index := range servedUEs {
		ue := servedUEs[index]
		ue.Cell.BwpRefs = []*model.Bwp{}
	}

	s.generateUsedPRBsIfMissing(availBWDL, availBWUL, scsOptions)

	sumCQIs := 0.0
	for _, ue := range servedUEs {
		sumCQIs += float64(ue.FiveQi)
	}

	for cqi, cqiStats := range s.StatsPerCQI {

		availBWDLCQI := int((float64(cqi * cqiStats.NumUEs * availBWDL)) / sumCQIs)
		availBWULCQI := int((float64(cqi * cqiStats.NumUEs * availBWUL)) / sumCQIs)

		cqiBwps := generateBWPs(availBWDLCQI, cqiStats.UsedPRBsDL, scsOptions)
		cqiBwpsUL := generateBWPs(availBWULCQI, cqiStats.UsedPRBsUL, scsOptions)

		lenCqiBwps := len(cqiBwps)
		for key, ulBwp := range cqiBwpsUL {
			newKey := lenCqiBwps + key
			ulBwp.ID = strconv.Itoa(newKey)
			cqiBwps[newKey] = ulBwp
		}

		allocateBWPsToUEs(cqiBwps, servedUEs, cqi)
	}

}

func (s *ProportionalFair) generateUsedPRBsIfMissing(availBWDL int, availBWUL int, scsOptions []int) {
	sumPRBsDL := 0
	sumPRBsUL := 0
	for _, cqiStats := range s.StatsPerCQI {
		sumPRBsDL += cqiStats.UsedPRBsDL
		sumPRBsUL += cqiStats.UsedPRBsUL
	}
	if sumPRBsDL == 0 {
		numUEsPerCQI := map[int]int{}
		for cqi, cqiStats := range s.StatsPerCQI {
			numUEsPerCQI[cqi] = cqiStats.NumUEs
		}

		usedBWDL := float64(availBWDL)
		// BWprb := 12 * SCSprb
		usedPRBsDL := int(usedBWDL / (12.0 * float64(scsOptions[0])))
		usedPRBsDLPerCQI := DisagregateCellUsedPRBs(numUEsPerCQI, usedPRBsDL)

		for cqi := range s.StatsPerCQI {
			cqiStats := s.StatsPerCQI[cqi]
			cqiStats.UsedPRBsDL = usedPRBsDLPerCQI[cqi]
			s.StatsPerCQI[cqi] = cqiStats
		}
	}

	if sumPRBsUL == 0 {
		numUEsPerCQI := map[int]int{}
		for cqi, cqiStats := range s.StatsPerCQI {
			numUEsPerCQI[cqi] = cqiStats.NumUEs
		}
		usedBWUL := float64(availBWUL)

		// BWprb := 12 * SCSprb
		usedPRBsUL := int(usedBWUL / (12.0 * float64(scsOptions[0])))
		usedPRBsULPerCQI := DisagregateCellUsedPRBs(numUEsPerCQI, usedPRBsUL)

		for cqi := range s.StatsPerCQI {
			cqiStats := s.StatsPerCQI[cqi]
			cqiStats.UsedPRBsUL = usedPRBsULPerCQI[cqi]
			s.StatsPerCQI[cqi] = cqiStats
		}
	}
}

func generateBWPs(remaingBW int, usedPRBs int, scsOptions []int) (cqiBwps map[int]model.Bwp) {
	cqiBwps = make(map[int]model.Bwp)
	lastSCSIndex := make(map[int]int)
	for remaingBW > 0 {
		for i := 0; i < usedPRBs; i++ {
			if remaingBW-12*scsOptions[lastSCSIndex[i]] > 0 {
				cqiBwps[i] = model.Bwp{
					ID:          strconv.Itoa(i),
					Scs:         scsOptions[lastSCSIndex[i]],
					NumberOfRBs: 1,
				}
				remaingBW -= 12 * scsOptions[lastSCSIndex[i]]
				lastSCSIndex[i]++
			}
		}
	}

	return
}

func allocateBWPsToUEs(cqiBwps map[int]model.Bwp, servedUEs []*model.UE, cqi int) {
	bwpsToAllocate := len(cqiBwps)
	for bwpsToAllocate > 0 {
		for index := range servedUEs {
			ue := servedUEs[index]
			if ue.FiveQi == cqi {
				bwp := cqiBwps[len(cqiBwps)-bwpsToAllocate]
				ue.Cell.BwpRefs = append(ue.Cell.BwpRefs, &bwp)
				bwpsToAllocate--
			}
		}
	}
}

func (s *ProportionalFair) reallocateBW(servedUEs []*model.UE, availBWDL int, availBWUL int) {
	ueRatesDL, ueRatesUL := s.getUeRates(servedUEs)
ALLOCATION:
	for _, ue := range servedUEs {
		log.Infof("UE %v: ueBWPercDL = %.2f, ueBWPercUL = %.2f\n", ue.IMSI, ueRatesDL[ue.IMSI], ueRatesUL[ue.IMSI])

		allocatedBWPsDL, err := s.reallocateBWPs(availBWDL, ueRatesDL[ue.IMSI], ue, true)
		if err != nil {
			log.Warnf("could not allocate downlink bw for ue: %v, %v", ue.IMSI, err)
			break ALLOCATION
		}
		ue.Cell.BwpRefs = append(ue.Cell.BwpRefs, allocatedBWPsDL...)

		allocatedBWPsUL, err := s.reallocateBWPs(availBWUL, ueRatesUL[ue.IMSI], ue, false)
		if err != nil {
			log.Warnf("could not allocate uplink bw for ue: %v, %v", ue.IMSI, err)
			break ALLOCATION
		}
		ue.Cell.BwpRefs = append(ue.Cell.BwpRefs, allocatedBWPsUL...)

		log.Infof("Assigned BWPs to UE %v (Downlink + Uplink): %v\n", ue.IMSI, ue.Cell.BwpRefs)
	}
}

func (s *ProportionalFair) getUeRates(servedUEs []*model.UE) (ueRatesDL, ueRatesUL map[types.IMSI]float64) {
	ueRatesDL = make(map[types.IMSI]float64)
	ueRatesUL = make(map[types.IMSI]float64)
	cellRequestedBWDL := 0.0
	cellRequestedBWUL := 0.0

	for _, ue := range servedUEs {
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

	for _, ue := range servedUEs {
		ueRatesDL[ue.IMSI] = ueRatesDL[ue.IMSI] / cellRequestedBWDL
		ueRatesDL[ue.IMSI] = ueRatesUL[ue.IMSI] / cellRequestedBWUL
	}

	return
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

// DisagregateCellUes only when no CQI Indexed Metrics exist and the Cell Metric exists.
// If CQI Indexed Metrics exist then, use them and ignore Cell Metric
func DisagregateCellUes(numUEsByCell map[uint64]map[string]int) map[uint64]map[int]int {

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
