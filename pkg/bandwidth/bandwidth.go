package bandwidth

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	log "github.com/sirupsen/logrus"
)

func InitBWPs(sCell *model.Cell, cellPrbMeas map[string]int, ues []*model.UE) error {

	// Existing BWPs from topology
	existingCellBwps := map[string]*model.Bwp{}
	if len(sCell.Bwps) != 0 {
		for index := range sCell.Bwps {
			bwp := *sCell.Bwps[index]
			existingCellBwps[bwp.ID] = &bwp
		}
		return nil
	}

	AllocateBW(sCell, cellPrbMeas, ues)

	if len(sCell.Bwps) == 0 {
		err := fmt.Errorf("failed to initialize BWPs for simulation")
		log.Error(err)
		return err
	}

	return nil

}

func ReleaseBWPs(sCell *model.Cell, ue *model.UE) []model.Bwp {
	bwps := make([]model.Bwp, 0, len(ue.Cell.BwpRefs))
	for index := range ue.Cell.BwpRefs {
		bwp := ue.Cell.BwpRefs[index]
		bwps = append(bwps, *bwp)
		delete(sCell.Bwps, bwp.ID)
	}
	ue.Cell.BwpRefs = []*model.Bwp{}
	return bwps
}

func ReallocateBW(ue *model.UE, requestedBwps []model.Bwp, tCell *model.Cell, servedUEs []*model.UE) {

	if enoughBW(tCell, requestedBwps) {
		bwpId := len(tCell.Bwps)
		for index := range requestedBwps {
			bwp := requestedBwps[index]
			bwp.ID = strconv.Itoa(bwpId)
			ue.Cell.BwpRefs = append(ue.Cell.BwpRefs, &bwp)
			tCell.Bwps[bwp.ID] = &bwp
			bwpId++
		}
		return
	}

	currAlloc := BwAllocationOf(servedUEs)

	// delete current allocation
	servedUEs = append(servedUEs, ue)
	for _, servedUe := range servedUEs {
		ReleaseBWPs(tCell, servedUe)
	}

	// reallocate using selected scheme
	switch tCell.ResourceAllocScheme {
	case PROPORTIONAL_FAIR:
		pf := ProportionalFair{
			PrevBwAllocation: currAlloc,
			ReqBwAllocation:  BwAllocationOf(servedUEs),
		}
		pf.apply(tCell, servedUEs)
	}
}

func AllocateBW(cell *model.Cell, cellPrbMeas map[string]int, servedUEs []*model.UE) {
	// Infer BWP allocation from cell prb measurements
	// pick used prbs if found else resort to total available
	cellUsedPRBsDLPerCQI, cellUsedPRBsULPerCQI := getUsedPRBsPerCQI(cellPrbMeas)

	// reallocate using selected scheme
	switch cell.ResourceAllocScheme {
	case PROPORTIONAL_FAIR:
		pf := ProportionalFair{
			UsedPRBsDLPerCQI: cellUsedPRBsDLPerCQI,
			UsedPRBsULPerCQI: cellUsedPRBsULPerCQI,
			TotalPRBsDL:      cellPrbMeas[TOTAL_PRBS_DL_METRIC],
			TotalPRBsUL:      cellPrbMeas[TOTAL_PRBS_DL_METRIC],
		}
		pf.apply(cell, servedUEs)
	}

}

func getUsedPRBsPerCQI(cellPrbMeas map[string]int) (map[int]int, map[int]int) {
	cellUsedPRBsDL := map[int]int{}
	cellUsedPRBsUL := map[int]int{}
	for metricName, numPrbs := range cellPrbMeas {
		cqi, err := strconv.Atoi(strings.Split(metricName, ".")[2])
		if err != nil {
			log.Errorf("Error converting CQI level to integer: %v", err)
			continue
		}
		switch {
		case MatchesPattern(metricName, USED_PRBS_DL_PATTERN):
			cellUsedPRBsDL[cqi] = numPrbs

		case MatchesPattern(metricName, USED_PRBS_UL_PATTERN):
			cellUsedPRBsUL[cqi] = numPrbs
		}
	}
	return cellUsedPRBsDL, cellUsedPRBsUL
}

func enoughBW(tCell *model.Cell, requestedBwps []model.Bwp) bool {
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
