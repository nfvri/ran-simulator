package bandwidth

import (
	"strconv"

	"github.com/nfvri/onos-api/go/onos/ransim/metrics"
	"github.com/nfvri/onos-api/go/onos/ransim/types"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"

	log "github.com/sirupsen/logrus"
)

const (
	PROPORTIONAL_FAIR = "PF"
	ROUND_ROBIN       = "RR"

	PRBS_UTIL_DL_METRIC = "RRU.PrbUtilDl"
	PRBS_UTIL_UL_METRIC = "RRU.PrbUtilUl"

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
	NumUEs          map[int]int
	UsedPRBsDL      map[int]int
	UsedPRBsUL      map[int]int
	AvailPRBsDL     int
	AvailPRBsUL     int
	IsReallocation  bool
	ReqBwAllocation map[types.IMSI][]model.Bwp
	Cell            *model.Cell
	ServedUEs       []*model.UE
	ScsOptionsHz    []int
}

// apply applies Proportional Fair scheduling to assign BWPs to UEs for both downlink and uplink
// ensuring total bandwidth limits are respected
func (s *ProportionalFair) apply() {

	if len(s.ScsOptionsHz) == 0 {
		pcc := s.Cell.Carriers[0]
		arfcn := utils.If(pcc.ArfcnDL > 0, pcc.ArfcnDL, pcc.ArfcnUL)
		direction := utils.If(pcc.ArfcnDL > 0, DL, UL)
		fr := GetFR(float64(arfcn))
		band, _ := GetBandNR(arfcn, direction)
		s.ScsOptionsHz = SupportedSCSByFR[fr]
		log.Infof(
			`fr: %s, band: %s, arfcnDL:%v, arfcnUL:%v, scs:%v`,
			fr, band.Name, pcc.ArfcnDL, pcc.ArfcnUL, s.ScsOptionsHz,
		)
	}

	totalBWDL := 0.0
	totalBWUL := 0.0
	for _, carrier := range s.Cell.Carriers {
		totalBWDL += MHzToHz(float64(carrier.BsChannelBwDL))
		totalBWUL += MHzToHz(float64(carrier.BsChannelBwUL))
	}

	availBWDL := int(totalBWDL * DEFAULT_MAX_BW_UTILIZATION)
	availBWUL := int(totalBWUL * DEFAULT_MAX_BW_UTILIZATION)

	if availBWDL == 0 && availBWUL == 0 {
		log.Warn("[PF] No bandwidth available for allocation.")
		return
	}

	if s.IsReallocation {
		log.Warn("[PF] Existing allocation found")
		log.Debugf("availBWDL:%v, availBWUL:%v", float64(availBWDL)/1e6, float64(availBWUL)/1e6)
		s.reallocateBW(availBWDL, availBWUL)
		return
	}

	sumUsedPRBsDL := 0
	for _, usedPRBs := range s.UsedPRBsDL {
		sumUsedPRBsDL += usedPRBs
	}
	sumUsedPRBsUL := 0
	for _, usedPRBs := range s.UsedPRBsUL {
		sumUsedPRBsUL += usedPRBs
	}

	// Update available BW based on current utilization
	if s.AvailPRBsDL != 0 && sumUsedPRBsDL != 0 {
		utilizationDL := float64(sumUsedPRBsDL) / float64(s.AvailPRBsDL)
		availBWDL = int(utilizationDL * float64(availBWDL))
	}
	if s.AvailPRBsUL != 0 && sumUsedPRBsUL != 0 {
		utilizationUL := float64(sumUsedPRBsUL) / float64(s.AvailPRBsUL)
		availBWUL = int(utilizationUL * float64(availBWUL))
	}

	if len(s.UsedPRBsDL) == 0 {
		s.generateUsedPRBs(availBWDL, true)
	}
	if len(s.UsedPRBsUL) == 0 {
		s.generateUsedPRBs(availBWUL, false)
	}
	log.Infof("--------------------")
	log.Infof("[PF] ncgi: %v", s.Cell.NCGI)
	log.Infof("[PF] availBWDL: %v", availBWDL)
	log.Infof("[PF] sumUsedPRBsDL: %v", sumUsedPRBsDL)
	log.Infof("[PF] availBWUL: %v", availBWUL)
	log.Infof("[PF] sumUsedPRBsUL: %v", sumUsedPRBsUL)
	log.Infof("--------------------")
	s.allocateBW(availBWDL, availBWUL)

}

func (s *ProportionalFair) allocateBW(availBWDL, availBWUL int) {

	for index := range s.ServedUEs {
		ue := s.ServedUEs[index]
		ueCell, _ := ue.GetServingCell(s.Cell.NCGI)
		ueCell.BwpRefs = []*model.Bwp{}
	}

	s.Cell.Bwps = map[uint64]*model.Bwp{}

	sumCQIs := 0.0
	for _, ue := range s.ServedUEs {
		sumCQIs += float64(ue.FiveQi)
	}

	remainingBWDl := 0
	remainingBWUl := 0

	for cqi, numUEs := range s.NumUEs {

		availBWDLCQI := int((float64(cqi * numUEs * availBWDL)) / sumCQIs)
		availBWULCQI := int((float64(cqi * numUEs * availBWUL)) / sumCQIs)

		usedPRBsDL, exixtsDL := s.UsedPRBsDL[cqi]
		if !exixtsDL {
			usedPRBsDL = 0
		}
		cqiBwpsDL, cqiRemaingBWDL := generateBWPs(availBWDLCQI+remainingBWDl, usedPRBsDL, true, s.ScsOptionsHz)

		usedPRBsUL, exixtsDL := s.UsedPRBsUL[cqi]
		if !exixtsDL {
			usedPRBsUL = 0
		}
		cqiBwpsUL, cqiRemaingBWUL := generateBWPs(availBWULCQI+remainingBWUl, usedPRBsUL, false, s.ScsOptionsHz)

		remainingBWDl = cqiRemaingBWDL
		remainingBWUl = cqiRemaingBWUL

		cqiBwps := append(cqiBwpsDL, cqiBwpsUL...)
		cellAllocatedBwps := len(s.Cell.Bwps)
		for i := range cqiBwps {
			bwp := cqiBwps[i]
			bwp.ID = uint64(cellAllocatedBwps + i)
			s.Cell.Bwps[bwp.ID] = bwp
		}
		allocateBWPsToUEs(cqiBwps, s.ServedUEs, cqi)
	}

	s.allocateRemainingBW(remainingBWDl, true)
	s.allocateRemainingBW(remainingBWUl, false)

}

func (s *ProportionalFair) allocateRemainingBW(remainingBW int, downlink bool) {
	if remainingBW > 12*s.ScsOptionsHz[0] {
		cqiBwps, _ := generateBWPs(remainingBW, 1, downlink, s.ScsOptionsHz)

		bwp := cqiBwps[0]
		bwp.ID = uint64(len(s.Cell.Bwps))
		s.Cell.Bwps[bwp.ID] = bwp

		maxCQI := 1
		maxNumUEs := s.NumUEs[maxCQI]
		for cqi, numUEs := range s.NumUEs {
			if numUEs > maxNumUEs && s.UsedPRBsDL[cqi] > 0 {
				maxNumUEs = numUEs
				maxCQI = cqi
			}
		}

		allocateBWPsToUEs(cqiBwps, s.ServedUEs, maxCQI)
	}
}

func (s *ProportionalFair) generateUsedPRBs(availBWHz int, downlink bool) {

	usedBWHz := float64(availBWHz)
	// BWprb := 12 * SCSprb
	usedPRBs := int(usedBWHz / float64(12*s.ScsOptionsHz[0]))
	usedPRBsPerCQI := DistributeCellUsedPRBsToCQIs(s.NumUEs, usedPRBs)

	if downlink {
		for cqi, usedPRBs := range usedPRBsPerCQI {
			s.UsedPRBsDL[cqi] = usedPRBs
		}
		return
	}
	for cqi, usedPRBs := range usedPRBsPerCQI {
		s.UsedPRBsUL[cqi] = usedPRBs
	}
}

func generateBWPs(remaingBWHz, usedPRBs int, downlink bool, scsOptions []int) ([]*model.Bwp, int) {
	cqiBwps := []*model.Bwp{}

	if usedPRBs == 0 {
		return cqiBwps, remaingBWHz
	}

	lastSCSIndex := make(map[int]int)

BW_PARTITION:
	for remaingBWHz > 0 {
		for i := 0; i < usedPRBs; i++ {
			if lastSCSIndex[i] == len(scsOptions) {
				break BW_PARTITION
			}
			if remaingBWHz-int(12*scsOptions[lastSCSIndex[i]]) < 0 {
				break BW_PARTITION

			}
			cqiBwps = append(cqiBwps, &model.Bwp{
				ID:          uint64(i),
				Scs:         scsOptions[lastSCSIndex[i]],
				NumberOfRBs: 1,
				Downlink:    downlink,
			})
			remaingBWHz -= 12 * scsOptions[lastSCSIndex[i]]
			lastSCSIndex[i]++
		}
	}

	return cqiBwps, remaingBWHz
}

func allocateBWPsToUEs(cqiBwps []*model.Bwp, servedUEs []*model.UE, cqi int) {
	bwpsToAllocate := len(cqiBwps)
BW_ALLOCATION:
	for bwpsToAllocate > 0 {
		startingBwp := bwpsToAllocate
		for index := range servedUEs {
			if bwpsToAllocate == 0 {
				break BW_ALLOCATION
			}
			ue := servedUEs[index]
			uePCell := ue.ServingCells[0]
			if ue.FiveQi == cqi {
				bwp := *cqiBwps[len(cqiBwps)-bwpsToAllocate]
				uePCell.BwpRefs = append(uePCell.BwpRefs, &bwp)
				bwpsToAllocate--
			}
		}
		if startingBwp == bwpsToAllocate {
			break BW_ALLOCATION
		}
	}
}

func (s *ProportionalFair) reallocateBW(availBWDL int, availBWUL int) {
	ueRatesDL, ueRatesUL := s.getUeRates()
	s.Cell.Bwps = map[uint64]*model.Bwp{}
	remainingBWDLHz := 0
	remainingBWULHz := 0

	for index := range s.ServedUEs {
		ue := s.ServedUEs[index]
		uePCell := ue.ServingCells[0]
		uePCellBwps := []model.Bwp{}

		if ueRateDL, ok := ueRatesDL[ue.IMSI]; ok {
			ueAvailBWDL := int(float64(availBWDL)*ueRateDL) + remainingBWDLHz
			allocatedBwpsDL, ueRemainingBWDLHz := s.reallocateBWPs(ueAvailBWDL, ue.IMSI, true)
			uePCellBwps = append(uePCellBwps, allocatedBwpsDL...)
			remainingBWDLHz = ueRemainingBWDLHz
		}
		if ueRateUL, ok := ueRatesUL[ue.IMSI]; ok {
			ueAvailBWUL := int(float64(availBWUL)*ueRateUL) + remainingBWULHz
			allocatedBWPsUL, ueRemainingBWULHz := s.reallocateBWPs(ueAvailBWUL, ue.IMSI, false)
			uePCellBwps = append(uePCellBwps, allocatedBWPsUL...)
			remainingBWULHz = ueRemainingBWULHz
		}

		if len(uePCellBwps) > 0 {
			cellAllocatedBwps := len(s.Cell.Bwps)
			for i := range uePCellBwps {
				bwp := uePCellBwps[i]
				bwp.ID = uint64(cellAllocatedBwps + i)
				s.Cell.Bwps[bwp.ID] = &bwp
				ueBWP := bwp
				uePCell.BwpRefs = append(uePCell.BwpRefs, &ueBWP)
			}
			// log.Infof("Assigned BWPs to UE %v (Downlink + Uplink): %v\n", ue.IMSI, len(ue.Cell.BwpRefs))
		}

	}
}

func (s *ProportionalFair) getUeRates() (ueRatesDL, ueRatesUL map[types.IMSI]float64) {
	ueRatesDL = map[types.IMSI]float64{}
	ueRatesUL = map[types.IMSI]float64{}

	uePeqBWDL := map[types.IMSI]float64{}
	ueReqBWUL := map[types.IMSI]float64{}
	cellRequestedBWDL := 0.0
	cellRequestedBWUL := 0.0

	for _, ue := range s.ServedUEs {
		ueReqBWPs, ok := s.ReqBwAllocation[ue.IMSI]
		if ok {
			for index := range ueReqBWPs {
				bwp := ueReqBWPs[index]
				if bwp.Downlink {
					uePeqBWDL[ue.IMSI] += float64(bwp.NumberOfRBs) * float64(bwp.Scs) * 12
				} else {
					ueReqBWUL[ue.IMSI] += float64(bwp.NumberOfRBs) * float64(bwp.Scs) * 12
				}
			}
		}

		if ueReqDL, ok := uePeqBWDL[ue.IMSI]; ok {
			cellRequestedBWDL += ueReqDL
		}

		if ueReqUL, ok := ueReqBWUL[ue.IMSI]; ok {
			cellRequestedBWUL += ueReqUL
		}
	}

	for _, ue := range s.ServedUEs {
		if ueReqDL, ok := uePeqBWDL[ue.IMSI]; ok {
			ueRatesDL[ue.IMSI] = ueReqDL / cellRequestedBWDL
		}
		if ueReqUL, ok := ueReqBWUL[ue.IMSI]; ok {
			ueRatesUL[ue.IMSI] = ueReqUL / cellRequestedBWUL
		}
	}
	return
}

// reallocateBWPs adjusts the BWPs for the UE based on available bandwidth
func (s *ProportionalFair) reallocateBWPs(availBWHz int, imsi types.IMSI, downlink bool) ([]model.Bwp, int) {

	remaingBWHz := float64(availBWHz)
	newBWPs := []model.Bwp{}
	requestedBWPs, ok := s.ReqBwAllocation[imsi]
	if !ok {
		return newBWPs, int(remaingBWHz)
	}
	reqBW := 0.0
	for i := range requestedBWPs {
		bwp := requestedBWPs[i]
		bwToAllocate := 12 * float64(bwp.NumberOfRBs) * float64(bwp.Scs)
		reqBW += bwToAllocate
		if remaingBWHz >= bwToAllocate {
			newBWPs = append(newBWPs, model.Bwp{
				ID:          uint64(i),
				Scs:         bwp.Scs,
				NumberOfRBs: int(bwp.NumberOfRBs),
				Downlink:    downlink,
			})
			remaingBWHz -= bwToAllocate
		}
	}

	if reqBW <= float64(availBWHz) {
		return newBWPs, int(remaingBWHz)
	}

	minPRBSize := 12 * float64(s.ScsOptionsHz[0])
	if remaingBWHz < minPRBSize {
		return newBWPs, int(remaingBWHz)
	}

	prbsToAllocate := int(remaingBWHz / minPRBSize)
	if prbsToAllocate > 0 {
		bwToAllocate := float64(prbsToAllocate) * minPRBSize
		newBWPs = append(newBWPs, model.Bwp{
			ID:          uint64(len(newBWPs)),
			Scs:         s.ScsOptionsHz[0],
			NumberOfRBs: prbsToAllocate,
			Downlink:    downlink,
		})
		remaingBWHz -= bwToAllocate
	}

	return newBWPs, int(remaingBWHz)
}

// ReallocateUsedPRBs only when both the Cell Metric and CQI Indexed Metrics exist.
// If Cell Metric doesn't exist, then use the CQI Indexed Metrics and don't ReallocateUsedPRBs
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
