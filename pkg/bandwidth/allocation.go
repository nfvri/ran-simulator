package bandwidth

import (
	"fmt"
	"math"
	"strconv"

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
	apply()
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
	Cell             *model.Cell
	ServedUEs        []*model.UE
	ScsOptionsHz     []int
}

// apply applies Proportional Fair scheduling to assign BWPs to UEs for both downlink and uplink
// ensuring total bandwidth limits are respected
func (s *ProportionalFair) apply() {

	if len(s.ScsOptionsHz) == 0 {
		s.ScsOptionsHz = []int{15_000, 30_000, 60_000, 120_000}
	}

	existingAllocation := len(s.PrevBwAllocation) > 0
	totalBWDL := MHzToHz(float64(s.Cell.Channel.BsChannelBwDL))
	totalBWUL := MHzToHz(float64(s.Cell.Channel.BsChannelBwUL))

	availBWDL := int(totalBWDL * DEFAULT_MAX_BW_UTILIZATION)
	availBWUL := int(totalBWUL * DEFAULT_MAX_BW_UTILIZATION)

	if s.AvailPRBsDL != 0 {
		availBWDL = s.AvailPRBsDL
	}

	if s.AvailPRBsUL != 0 {
		availBWUL = s.AvailPRBsUL
	}

	if availBWDL == 0 && availBWUL == 0 {
		fmt.Println("No bandwidth available for allocation.")
		return
	}

	if existingAllocation {
		s.reallocateBW(availBWDL, availBWUL)
		return
	}

	s.allocateBW(availBWDL, availBWUL)

}

func (s *ProportionalFair) allocateBW(availBWDL, availBWUL int) {

	for index := range s.ServedUEs {
		ue := s.ServedUEs[index]
		ue.Cell.BwpRefs = []*model.Bwp{}
	}

	s.generateUsedPRBsIfMissing(availBWDL, availBWUL)

	sumCQIs := 0.0
	for _, ue := range s.ServedUEs {
		sumCQIs += float64(ue.FiveQi)
	}

	remainingBWDl := 0
	remainingBWUl := 0

	for cqi, cqiStats := range s.StatsPerCQI {

		availBWDLCQI := int((float64(cqi * cqiStats.NumUEs * availBWDL)) / sumCQIs)
		availBWULCQI := int((float64(cqi * cqiStats.NumUEs * availBWUL)) / sumCQIs)

		cqiBwpsDL, cqiRemaingBWDL := generateBWPs(availBWDLCQI+remainingBWDl, cqiStats.UsedPRBsDL)
		cqiBwpsUL, cqiRemaingBWUL := generateBWPs(availBWULCQI+remainingBWUl, cqiStats.UsedPRBsUL)

		remainingBWDl = cqiRemaingBWDL
		remainingBWUl = cqiRemaingBWUL

		cqiBwps := append(cqiBwpsDL, cqiBwpsUL...)
		cellAllocatedBwps := len(s.Cell.Bwps)
		for i := range cqiBwps {
			bwp := *cqiBwps[i]
			bwp.ID = uint64(cellAllocatedBwps + i)
			s.Cell.Bwps[bwp.ID] = &bwp
		}

		allocateBWPsToUEs(cqiBwps, s.ServedUEs, cqi)
	}

}

func (s *ProportionalFair) generateUsedPRBsIfMissing(availBWDLHz int, availBWULHz int) {
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

		usedBWDLHz := float64(availBWDLHz)
		// BWprb := 12 * SCSprb
		usedPRBsDL := int(usedBWDLHz / float64(12*s.ScsOptionsHz[0]))
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
		usedBWULHz := float64(availBWULHz)

		// BWprb := 12 * SCSprb
		usedPRBsUL := int(usedBWULHz / (12.0 * float64(s.ScsOptionsHz[0])))
		usedPRBsULPerCQI := DisagregateCellUsedPRBs(numUEsPerCQI, usedPRBsUL)

		for cqi := range s.StatsPerCQI {
			cqiStats := s.StatsPerCQI[cqi]
			cqiStats.UsedPRBsUL = usedPRBsULPerCQI[cqi]
			s.StatsPerCQI[cqi] = cqiStats
		}
	}
}

func generateBWPs(remaingBWHz int, usedPRBs int) ([]*model.Bwp, int) {
	scsOptions := []int{15_000, 30_000, 60_000, 120_000}
	cqiBwps := []*model.Bwp{}
	lastSCSIndex := make(map[int]int)

BW_PARTITION:
	for remaingBWHz > 0 {
		for i := 0; i < usedPRBs; i++ {
			if remaingBWHz-int(12*scsOptions[lastSCSIndex[i]]) < 0 {
				break BW_PARTITION
			}
			cqiBwps = append(cqiBwps, &model.Bwp{
				ID:          uint64(i),
				Scs:         scsOptions[lastSCSIndex[i]],
				NumberOfRBs: 1,
			})
			remaingBWHz -= 12 * scsOptions[lastSCSIndex[i]]
			lastSCSIndex[i]++
		}
	}

	return cqiBwps, remaingBWHz
}

func allocateBWPsToUEs(cqiBwps []*model.Bwp, servedUEs []*model.UE, cqi int) {
	bwpsToAllocate := len(cqiBwps)
	for bwpsToAllocate > 0 {
		for index := range servedUEs {
			ue := servedUEs[index]
			if ue.FiveQi == cqi {
				bwp := *cqiBwps[len(cqiBwps)-bwpsToAllocate]
				ue.Cell.BwpRefs = append(ue.Cell.BwpRefs, &bwp)
				bwpsToAllocate--
			}
		}
	}
}

func (s *ProportionalFair) reallocateBW(availBWDL int, availBWUL int) {
	ueRatesDL, ueRatesUL := s.getUeRates()
	s.Cell.Bwps = map[uint64]*model.Bwp{}
	remainingBWDLHz := 0
	remainingBWULHz := 0
	ueCellBwps := []*model.Bwp{}
	for _, ue := range s.ServedUEs {
		log.Infof("UE %v: ueBWPercDL = %.2f, ueBWPercUL = %.2f\n", ue.IMSI, ueRatesDL[ue.IMSI], ueRatesUL[ue.IMSI])

		allocatedBWPsDL, ueRemainingBWDLHz := s.reallocateBWPs(availBWDL+remainingBWDLHz, ueRatesDL[ue.IMSI], ue.IMSI, true)
		remainingBWDLHz = ueRemainingBWDLHz

		ueCellBwps = append(ueCellBwps, allocatedBWPsDL...)

		allocatedBWPsUL, ueRemainingBWULHz := s.reallocateBWPs(availBWUL+remainingBWULHz, ueRatesUL[ue.IMSI], ue.IMSI, false)
		remainingBWULHz = ueRemainingBWULHz

		ueCellBwps = append(ueCellBwps, allocatedBWPsUL...)

		cellAllocatedBwps := len(s.Cell.Bwps)
		for i := range ueCellBwps {
			bwp := *ueCellBwps[i]
			bwp.ID = uint64(cellAllocatedBwps + i)
			s.Cell.Bwps[bwp.ID] = &bwp
			ue.Cell.BwpRefs = append(ue.Cell.BwpRefs, &bwp)
		}

		log.Infof("Assigned BWPs to UE %v (Downlink + Uplink): %v\n", ue.IMSI, ue.Cell.BwpRefs)
	}
}

func (s *ProportionalFair) getUeRates() (ueRatesDL, ueRatesUL map[types.IMSI]float64) {
	ueRatesDL = make(map[types.IMSI]float64)
	ueRatesUL = make(map[types.IMSI]float64)
	cellRequestedBWDL := 0.0
	cellRequestedBWUL := 0.0

	for _, ue := range s.ServedUEs {
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

	for _, ue := range s.ServedUEs {
		ueRatesDL[ue.IMSI] = ueRatesDL[ue.IMSI] / cellRequestedBWDL
		ueRatesDL[ue.IMSI] = ueRatesUL[ue.IMSI] / cellRequestedBWUL
	}

	return
}

// reallocateBWPs adjusts the BWPs for the UE based on available bandwidth
func (s *ProportionalFair) reallocateBWPs(availBWHz int, ueRate float64, imsi types.IMSI, downlink bool) ([]*model.Bwp, int) {
	ueNewBWHz := int(float64(availBWHz) * ueRate)

	newBWPs := []*model.Bwp{}
	previousBWPs := s.PrevBwAllocation[imsi]
	remaingBWHz := float64(ueNewBWHz)

	for i, bwp := range previousBWPs {
		remainingPRBs := remaingBWHz / 12 * float64(bwp.Scs)
		rbsToAllocate := math.Min(remainingPRBs, float64(bwp.NumberOfRBs))
		// TODO: check on remainingBW
		//  try progressively lower scs when remainingBW >0 but newBWP+allocated > ueNewBW
		if remaingBWHz > 12*rbsToAllocate*float64(bwp.Scs) {
			newBWPs[i] = &model.Bwp{
				ID:          uint64(i),
				Scs:         bwp.Scs,
				NumberOfRBs: int(rbsToAllocate),
				Downlink:    downlink,
			}
			remaingBWHz -= float64(int(rbsToAllocate) * 12 * bwp.Scs)
		}
	}

	if remaingBWHz > 0 {
		bwpSizeHz := 12.0 * float64(s.ScsOptionsHz[0])
		remainingPRBs := int(remaingBWHz / bwpSizeHz)
		newBWPs[len(newBWPs)] = &model.Bwp{
			ID:          uint64(len(newBWPs)),
			Scs:         s.ScsOptionsHz[0],
			NumberOfRBs: remainingPRBs,
			Downlink:    downlink,
		}
		remaingBWHz -= float64(remainingPRBs) * 12 * float64(s.ScsOptionsHz[0])
	}

	return newBWPs, int(remaingBWHz)
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
