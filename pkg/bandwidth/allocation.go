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
	NumUEs           map[int]int
	UsedPRBsDlPerCQI map[int]int
	UsedPRBsUlPerCQI map[int]int
	AvailPRBsDL      int
	AvailPRBsUL      int
	IsReallocation   bool
	ReqBwAllocation  map[types.IMSI][]model.Bwp
	Cell             *model.Cell
	ServedUEs        []*model.UE
	ScsKHzPerCQI     map[int]int
}

// apply applies Proportional Fair scheduling to assign BWPs to UEs for both downlink and uplink
// ensuring total bandwidth limits are respected
func (s *ProportionalFair) apply() {

	s.setSCSOptions()

	totalBwDlHz := 0.0
	totalBwUlHz := 0.0
	for _, carrier := range s.Cell.Carriers {
		totalBwDlHz += MHzToHz(float64(carrier.BsChannelBwDL))
		totalBwUlHz += MHzToHz(float64(carrier.BsChannelBwUL))
	}

	// FIXME: use guardband for cell
	availBwDlHz := int(totalBwDlHz * DEFAULT_MAX_BW_UTILIZATION)
	availBwUlHz := int(totalBwUlHz * DEFAULT_MAX_BW_UTILIZATION)

	if availBwDlHz == 0 && availBwUlHz == 0 {
		log.Warnf("ncgi: %v | [PF] No bandwidth available for allocation.", s.Cell.NCGI)
		return
	}

	if s.IsReallocation {
		log.Warnf("ncgi: %v | [PF] Existing allocation found", s.Cell.NCGI)
		log.Debugf("availBWDL:%v, availBWUL:%v", float64(availBwDlHz)/1e6, float64(availBwUlHz)/1e6)
		s.reallocateBW(availBwDlHz, availBwUlHz)
		return
	}

	sumUsedPRBsDL := 0
	for _, usedPRBs := range s.UsedPRBsDlPerCQI {
		sumUsedPRBsDL += usedPRBs
	}
	sumUsedPRBsUL := 0
	for _, usedPRBs := range s.UsedPRBsUlPerCQI {
		sumUsedPRBsUL += usedPRBs
	}

	// Update available BW based on current utilization
	if s.AvailPRBsDL != 0 && sumUsedPRBsDL != 0 {
		utilizationDL := float64(sumUsedPRBsDL) / float64(s.AvailPRBsDL)
		availBwDlHz = int(utilizationDL * float64(availBwDlHz))
	}
	if s.AvailPRBsUL != 0 && sumUsedPRBsUL != 0 {
		utilizationUL := float64(sumUsedPRBsUL) / float64(s.AvailPRBsUL)
		availBwUlHz = int(utilizationUL * float64(availBwUlHz))
	}

	if len(s.UsedPRBsDlPerCQI) == 0 {
		s.populateUsedPRBs(availBwDlHz, true)
	}
	if len(s.UsedPRBsUlPerCQI) == 0 {
		s.populateUsedPRBs(availBwUlHz, false)
	}
	log.Infof("--------------------")
	log.Infof("NEW ALLOCATION")
	log.Infof("--------------------")
	log.Infof("[PF] ncgi: %v", s.Cell.NCGI)
	log.Infof("[PF] availBwDl_MHz: %v", HzToMHz(float64(availBwDlHz)))
	log.Infof("[PF] AvailPRBsDL: %v", s.AvailPRBsDL)
	log.Infof("[PF] sumUsedPRBsDL: %v", sumUsedPRBsDL)
	log.Infof("[PF] availBwUl_MHz: %v", HzToMHz(float64(availBwUlHz)))
	log.Infof("[PF] AvailPRBsUL: %v", s.AvailPRBsUL)
	log.Infof("[PF] sumUsedPRBsUL: %v", sumUsedPRBsUL)
	log.Infof("--------------------")
	s.allocateBW(availBwDlHz, availBwUlHz)

}

func (s *ProportionalFair) setSCSOptions() {
	if len(s.ScsKHzPerCQI) == 0 {
		pcc := s.Cell.Carriers[0]
		arfcn := utils.If(pcc.ArfcnDL > 0, pcc.ArfcnDL, pcc.ArfcnUL)
		direction := utils.If(pcc.ArfcnDL > 0, DL, UL)
		fr := GetFR(float64(arfcn))
		band, _ := GetBandNR(arfcn, direction)
		s.ScsKHzPerCQI = NrSCSByCQIPerFR[fr]
		log.Infof(
			`fr: %s, band: %s, arfcnDL:%v, arfcnUL:%v, scs:%+v`,
			fr, band.Name, pcc.ArfcnDL, pcc.ArfcnUL, s.ScsKHzPerCQI,
		)
	}
}

func (s *ProportionalFair) allocateBW(availBwDlHz, availBwUlHz int) {

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

	remainingBwDlHz := 0
	remainingBwUlHz := 0

	for cqi, numUEs := range s.NumUEs {

		cqiAvailBwDlHz := int((float64(cqi * numUEs * availBwDlHz)) / sumCQIs)
		cqiAvailBwUlHz := int((float64(cqi * numUEs * availBwUlHz)) / sumCQIs)

		usedPRBsDL, exixtsDL := s.UsedPRBsDlPerCQI[cqi]
		if !exixtsDL {
			usedPRBsDL = 0
		}
		cqiBwpsDL, cqiRemaingBwDlHz := generateBWPs(cqiAvailBwDlHz+remainingBwDlHz, usedPRBsDL, true, s.ScsKHzPerCQI[cqi])

		usedPRBsUL, exixtsDL := s.UsedPRBsUlPerCQI[cqi]
		if !exixtsDL {
			usedPRBsUL = 0
		}
		cqiBwpsUL, cqiRemaingBwUlHz := generateBWPs(cqiAvailBwUlHz+remainingBwUlHz, usedPRBsUL, false, s.ScsKHzPerCQI[cqi])

		remainingBwDlHz = cqiRemaingBwDlHz
		remainingBwUlHz = cqiRemaingBwUlHz

		cqiBwps := append(cqiBwpsDL, cqiBwpsUL...)
		cellAllocatedBwps := len(s.Cell.Bwps)
		for i := range cqiBwps {
			bwp := cqiBwps[i]
			bwp.ID = uint64(cellAllocatedBwps + i)
			s.Cell.Bwps[bwp.ID] = bwp
		}
		allocateBWPsToUEs(cqiBwps, s.ServedUEs, cqi)
	}

	s.allocateRemainingBW(remainingBwDlHz, true)
	s.allocateRemainingBW(remainingBwUlHz, false)

}

func (s *ProportionalFair) allocateRemainingBW(remainingBwHz int, downlink bool) {
	scsHz := KHzToHz(float64(s.ScsKHzPerCQI[15]))
	if float64(remainingBwHz) > 12*scsHz {
		prbsToGenerate := remainingBwHz / int(scsHz)
		cqiBwps, _ := generateBWPs(remainingBwHz, prbsToGenerate, downlink, s.ScsKHzPerCQI[15])

		bwp := cqiBwps[0]
		bwp.ID = uint64(len(s.Cell.Bwps))
		s.Cell.Bwps[bwp.ID] = bwp

		maxCQI := 1
		maxNumUEs := s.NumUEs[maxCQI]
		for cqi, numUEs := range s.NumUEs {
			if numUEs > maxNumUEs && s.UsedPRBsDlPerCQI[cqi] > 0 {
				maxNumUEs = numUEs
				maxCQI = cqi
			}
		}

		allocateBWPsToUEs(cqiBwps, s.ServedUEs, maxCQI)
	}
}

func (s *ProportionalFair) populateUsedPRBs(availBWHz int, downlink bool) {

	usedBWHz := float64(availBWHz)
	usedPRBs := int(usedBWHz / float64(12*s.ScsKHzPerCQI[15]))
	usedPRBsPerCQI := DistributeCellUsedPRBsToCQIs(s.NumUEs, usedPRBs)

	if downlink {
		for cqi, usedPRBs := range usedPRBsPerCQI {
			s.UsedPRBsDlPerCQI[cqi] = usedPRBs
		}
		return
	}
	for cqi, usedPRBs := range usedPRBsPerCQI {
		s.UsedPRBsUlPerCQI[cqi] = usedPRBs
	}
}

func generateBWPs(remaingBWHz, usedPRBs int, downlink bool, scsKHz int) ([]*model.Bwp, int) {
	cqiBwps := []*model.Bwp{}

	if usedPRBs == 0 {
		return cqiBwps, remaingBWHz
	}

	for i := 0; i < usedPRBs; i++ {
		scsHz := KHzToHz(float64(scsKHz))

		if remaingBWHz-int(12*scsHz) < 0 {
			break
		}

		numBwps := (scsKHz / 15)
		for j := 0; j < numBwps; j++ {
			cqiBwps = append(cqiBwps, &model.Bwp{
				ID:          uint64(i),
				Scs:         scsKHz,
				NumberOfRBs: 1,
				Downlink:    downlink,
			})
		}
		i += numBwps - 1
		remaingBWHz -= 12 * int(scsHz)
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
		bwToAllocate := 12 * float64(bwp.NumberOfRBs) * KHzToHz(float64(bwp.Scs))
		reqBW += bwToAllocate
		if remaingBWHz >= bwToAllocate {
			newBWPs = append(newBWPs, model.Bwp{
				ID:          uint64(i),
				Scs:         bwp.Scs,
				NumberOfRBs: bwp.NumberOfRBs,
				Downlink:    downlink,
			})
			remaingBWHz -= bwToAllocate
		}
	}

	if reqBW <= float64(availBWHz) {
		return newBWPs, int(remaingBWHz)
	}

	minPRBSize := 12 * KHzToHz(float64(s.ScsKHzPerCQI[15]))
	if remaingBWHz < minPRBSize {
		return newBWPs, int(remaingBWHz)
	}

	prbsToAllocate := int(remaingBWHz / minPRBSize)
	if prbsToAllocate > 0 {
		bwToAllocate := float64(prbsToAllocate) * minPRBSize
		newBWPs = append(newBWPs, model.Bwp{
			ID:          uint64(len(newBWPs)),
			Scs:         s.ScsKHzPerCQI[15],
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
