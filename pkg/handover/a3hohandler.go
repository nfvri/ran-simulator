package handover

import (
	"strconv"

	"github.com/nfvri/ran-simulator/pkg/model"
)

// NewA3HandoverHandler returns A3HandoverHandler object
func NewA3HandoverHandler(m *model.Model) *A3HandoverHandler {
	return &A3HandoverHandler{
		Chans: A3HandoverChannel{
			InputChan:  make(chan *model.UE),
			OutputChan: make(chan A3HandoverDecision),
		},
		Model: m,
	}
}

const MIN_ACCEPTABLE_RSRP = -110

// A3HandoverHandler is A3 handover handler
type A3HandoverHandler struct {
	Chans A3HandoverChannel
	Model *model.Model
}

// A3HandoverChannel struct has channels used in A3 handover handler
type A3HandoverChannel struct {
	InputChan  chan *model.UE
	OutputChan chan A3HandoverDecision
}

// Run starts A3 handover handler
func (h *A3HandoverHandler) Run() {
	for ue := range h.Chans.InputChan {
		tCell := h.getTargetCell(ue)
		h.Chans.OutputChan <- A3HandoverDecision{
			UE:          ue,
			ServingCell: h.Model.GetServingCells(ue.IMSI)[0],
			TargetCell:  tCell,
			Feasible:    h.isHOFeasible(tCell, ue),
		}
	}
}

func (h *A3HandoverHandler) isHOFeasible(tUECell *model.UECell, ue *model.UE) bool {

	tCellNcgiStr := strconv.FormatUint(uint64(tUECell.NCGI), 10)
	tCell := h.Model.Cells[tCellNcgiStr]

	//TODO: check if UL+DL is sufficient istead of individual checks
	requestedBWDL := 0
	requestedBWUL := 0

	for _, bwp := range ue.Cell.BwpRefs {
		if bwp.Downlink {
			requestedBWDL += bwp.Scs * 12 * bwp.NumberOfRBs
		} else {
			requestedBWUL += bwp.Scs * 12 * bwp.NumberOfRBs
		}
	}

	allocatedBWDL := 0
	allocatedBWUL := 0
	for _, bwp := range tUECell.BwpRefs {
		if bwp.Downlink {
			allocatedBWDL += bwp.Scs * 12 * bwp.NumberOfRBs
		} else {
			allocatedBWUL += bwp.Scs * 12 * bwp.NumberOfRBs
		}
	}

	sufficientBWDL := tCell.Channel.BsChannelBwDL-uint32(allocatedBWDL) > uint32(requestedBWDL)
	sufficientBWUL := tCell.Channel.BsChannelBwUL-uint32(allocatedBWUL) > uint32(requestedBWUL)

	return sufficientBWDL && sufficientBWUL && tUECell.Rsrp >= MIN_ACCEPTABLE_RSRP
}

func (h *A3HandoverHandler) getTargetCell(ue *model.UE) *model.UECell {
	targetCell := ue.Cell
	bestRSRP := ue.Cell.Rsrp

	for _, cscell := range ue.Cells {
		if cscell.Rsrp > bestRSRP {
			targetCell = cscell
			bestRSRP = cscell.Rsrp
		}
	}
	return targetCell
}

// A3HandoverDecision struct has A3 handover decision information
type A3HandoverDecision struct {
	UE          *model.UE
	ServingCell *model.Cell
	TargetCell  *model.UECell
	Feasible    bool
}
