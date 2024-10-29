package bandwidth

import (
	"sort"
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	"github.com/stretchr/testify/assert"
)

func Test_allocateBW_cqiProportionally(t *testing.T) {

	cell, servedUEs := setup()

	pf := ProportionalFair{
		StatsPerCQI: map[int]CQIStats{
			1: {
				NumUEs:     1,
				UsedPRBsDL: 20,
				UsedPRBsUL: 20,
			},
			8: {
				NumUEs:     1,
				UsedPRBsDL: 30,
				UsedPRBsUL: 30,
			},
			15: {
				NumUEs:     1,
				UsedPRBsDL: 60,
				UsedPRBsUL: 60,
			},
		},
		Cell:      cell,
		ServedUEs: servedUEs,
	}

	totalBWDL := MHzToHz(float64(cell.Channel.BsChannelBwDL))
	totalBWUL := MHzToHz(float64(cell.Channel.BsChannelBwUL))

	availBWDL := int(totalBWDL * DEFAULT_MAX_BW_UTILIZATION)
	availBWUL := int(totalBWUL * DEFAULT_MAX_BW_UTILIZATION)

	pf.allocateBW(availBWDL, availBWUL)

	verifyBwIncreasesWithCQI(t, servedUEs)
}

func Test_ProportionalFair_apply_allocation(t *testing.T) {
	cell, servedUEs := setup()

	pf := ProportionalFair{
		StatsPerCQI: map[int]CQIStats{
			1: {
				NumUEs:     1,
				UsedPRBsDL: 20,
				UsedPRBsUL: 20,
			},
			8: {
				NumUEs:     1,
				UsedPRBsDL: 30,
				UsedPRBsUL: 30,
			},
			15: {
				NumUEs:     1,
				UsedPRBsDL: 60,
				UsedPRBsUL: 60,
			},
		},
		Cell:      cell,
		ServedUEs: servedUEs,
	}

	pf.apply()

	verifyBwNotExceeded(t, cell, servedUEs)
	verifyBwIncreasesWithCQI(t, servedUEs)
}

func Test_ProportionalFair_apply_reallocation(t *testing.T) {
	cell, servedUEs := setup()
	cell.Channel.BsChannelBwDL = 2
	cell.Channel.BsChannelBwUL = 1

	existingAlloc := createCurrAlloc(cell, servedUEs)

	newUe := &model.UE{
		IMSI:   types.IMSI(6935566888),
		FiveQi: 8,
		Cell: &model.UECell{
			ID:   17680452419585,
			NCGI: 17680452419585,
			Rsrp: -88.84787974766571,
			Rsrq: 21.7074,
			Sinr: 4.909738334537388,
		},
	}

	newRequestedAlloc := map[types.IMSI][]model.Bwp{}
	for imsi, bwps := range existingAlloc {
		newRequestedAlloc[imsi] = bwps
	}

	newRequestedAlloc[newUe.IMSI] = []model.Bwp{
		{
			ID:          0,
			Scs:         15000,
			NumberOfRBs: 3,
			Downlink:    true,
		},
		{
			ID:          1,
			Scs:         15000,
			NumberOfRBs: 1,
			Downlink:    false,
		},
	}

	servedUEs = append(servedUEs, newUe)

	pf := ProportionalFair{
		Cell:             cell,
		ServedUEs:        servedUEs,
		ReqBwAllocation:  newRequestedAlloc,
		PrevBwAllocation: existingAlloc,
	}

	pf.apply()

	verifyBwNotExceeded(t, cell, servedUEs)
	verifyBwIncreasesWithCQI(t, servedUEs)
}

// ****************************************************
//
//	Helpers
//
// ****************************************************
func setup() (*model.Cell, []*model.UE) {
	cell := &model.Cell{
		NCGI: 1234,
		Bwps: make(map[uint64]*model.Bwp),
		CellConfig: model.CellConfig{
			Channel: model.Channel{
				SSBFrequency:   3400,
				ArfcnDL:        180000,
				ArfcnUL:        180000,
				Environment:    "urban",
				BsChannelBwDL:  40,
				BsChannelBwUL:  35,
				BsChannelBwSUL: 0,
				LOS:            false,
			},
		},
	}

	servedUEs := []*model.UE{
		{
			IMSI:   types.IMSI(6935566777),
			FiveQi: 1,
			Cell: &model.UECell{
				ID:   17680452419585,
				NCGI: 17680452419585,
				Rsrp: -88.84787974766571,
				Rsrq: 21.7074,
				Sinr: 4.909738334537388,
			},
		},
		{
			IMSI:   types.IMSI(6935566778),
			FiveQi: 8,
			Cell: &model.UECell{
				ID:   17680452419585,
				NCGI: 17680452419585,
				Rsrp: -88.84787974766571,
				Rsrq: 21.7074,
				Sinr: 4.909738334537388,
			},
		},
		{
			IMSI:   types.IMSI(6935566779),
			FiveQi: 15,
			Cell: &model.UECell{
				ID:   17680452419585,
				NCGI: 17680452419585,
				Rsrp: -88.84787974766571,
				Rsrq: 21.7074,
				Sinr: 4.909738334537388,
			},
		},
	}
	return cell, servedUEs
}

func createCurrAlloc(cell *model.Cell, servedUEs []*model.UE) map[types.IMSI][]model.Bwp {
	bwPartition := map[uint64]*model.Bwp{
		0: {
			ID:          0,
			Scs:         15000,
			NumberOfRBs: 1,
			Downlink:    true,
		},
		1: {
			ID:          1,
			Scs:         15000,
			NumberOfRBs: 1,
			Downlink:    true,
		},
		2: {
			ID:          2,
			Scs:         15000,
			NumberOfRBs: 2,
			Downlink:    true,
		},
		3: {
			ID:          3,
			Scs:         15000,
			NumberOfRBs: 1,
			Downlink:    true,
		},
		4: {
			ID:          4,
			Scs:         15000,
			NumberOfRBs: 3,
			Downlink:    true,
		},
		5: {
			ID:          5,
			Scs:         15000,
			NumberOfRBs: 2,
			Downlink:    true,
		},
		6: {
			ID:          6,
			Scs:         15000,
			NumberOfRBs: 2,
			Downlink:    false,
		},
		7: {
			ID:          7,
			Scs:         15000,
			NumberOfRBs: 1,
			Downlink:    false,
		},
		8: {
			ID:          8,
			Scs:         15000,
			NumberOfRBs: 2,
			Downlink:    false,
		},
	}

	cell.Bwps = bwPartition
	currAlloc := map[types.IMSI][]model.Bwp{}

	bwpToAllocate := len(bwPartition)
	for bwpToAllocate > 0 {
		for index := range servedUEs {
			if bwpToAllocate == 0 {
				break
			}
			ue := servedUEs[index]
			nextBwpIndex := len(bwPartition) - bwpToAllocate
			bwp := bwPartition[uint64(nextBwpIndex)]
			if _, ok := currAlloc[ue.IMSI]; !ok {
				currAlloc[ue.IMSI] = []model.Bwp{}
			}
			currAlloc[ue.IMSI] = append(currAlloc[ue.IMSI], *bwp)
			bwpToAllocate--
		}
	}
	return currAlloc
}

// ****************************************************
//
//	Verifiers
//
// ****************************************************
func verifyBwNotExceeded(t *testing.T, cell *model.Cell, servedUEs []*model.UE) {
	usedBWDL := 0
	usedBWUL := 0
	for _, ue := range servedUEs {
		ueUsedBWDL := 0
		ueUsedBWUL := 0
		for _, bwp := range ue.Cell.BwpRefs {
			if bwp.Downlink {
				ueUsedBWDL += 12 * bwp.NumberOfRBs * bwp.Scs
			} else {
				ueUsedBWUL += 12 * bwp.NumberOfRBs * bwp.Scs
			}
		}
		usedBWDL += ueUsedBWDL
		usedBWUL += ueUsedBWUL
		t.Logf("ue:%v usedBWDL: %v, usedBWUL: %v", ue.FiveQi, float64(ueUsedBWDL)/1e6, float64(ueUsedBWUL)/1e6)
	}
	assert.LessOrEqual(t, float64(usedBWDL)/1e6, float64(cell.Channel.BsChannelBwDL)*DEFAULT_MAX_BW_UTILIZATION)
	assert.LessOrEqual(t, float64(usedBWUL)/1e6, float64(cell.Channel.BsChannelBwUL)*DEFAULT_MAX_BW_UTILIZATION)
}

func verifyBwIncreasesWithCQI(t *testing.T, servedUEs []*model.UE) {
	sort.SliceStable(servedUEs, func(i, j int) bool {
		return servedUEs[i].FiveQi < servedUEs[j].FiveQi
	})
	bwAllocationDL := make([]float64, len(servedUEs))
	bwAllocationUL := make([]float64, len(servedUEs))
	for i, ue := range servedUEs {
		t.Log(ue.FiveQi)
		for _, bwp := range ue.Cell.BwpRefs {
			if bwp.Downlink {
				bwAllocationDL[i] += 12 * float64(bwp.NumberOfRBs) * float64(bwp.Scs)
			} else {
				bwAllocationUL[i] += 12 * float64(bwp.NumberOfRBs) * float64(bwp.Scs)
			}
		}
		if i > 0 {
			assert.LessOrEqual(t, servedUEs[i-1].FiveQi, servedUEs[i].FiveQi)
		}
	}

	t.Log(bwAllocationDL)
	t.Log(bwAllocationUL)

	for i, ue := range servedUEs {
		if i > 0 {
			if ue.FiveQi < servedUEs[i-1].FiveQi {
				assert.LessOrEqual(t, bwAllocationDL[i-1], bwAllocationDL[i])
				assert.LessOrEqual(t, bwAllocationUL[i-1], bwAllocationUL[i])
			}
		}
	}
}
