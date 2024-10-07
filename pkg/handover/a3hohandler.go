package handover

import (
	"github.com/nfvri/ran-simulator/pkg/model"
)

// NewA3HandoverHandler returns A3HandoverHandler object
func NewA3HandoverHandler(m *model.Model) *A3HandoverHandler {
	return &A3HandoverHandler{
		Chans: A3HandoverChannel{
			InputChan:  make(chan *model.UE),
			OutputChan: make(chan HandoverDecision),
		},
		Model: m,
	}
}

// A3HandoverHandler is A3 handover handler
type A3HandoverHandler struct {
	Chans A3HandoverChannel
	Model *model.Model
}

// A3HandoverChannel struct has channels used in A3 handover handler
type A3HandoverChannel struct {
	InputChan  chan *model.UE
	OutputChan chan HandoverDecision
}

// Run starts A3 handover handler
func (h *A3HandoverHandler) Run() {
	for ue := range h.Chans.InputChan {
		tCell := h.getTargetCell(ue)
		h.Chans.OutputChan <- HandoverDecision{
			UE:          ue,
			ServingCell: h.Model.GetServingCells(ue.IMSI)[0],
			TargetCell:  tCell,
		}
	}
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
