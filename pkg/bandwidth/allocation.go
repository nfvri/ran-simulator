package bandwidth

import (
	"fmt"
	"math"
	"sort"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	log "github.com/sirupsen/logrus"
)

const (
	PROPORTIONAL_FAIR = "PF"
	ROUND_ROBIN       = "RR"
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