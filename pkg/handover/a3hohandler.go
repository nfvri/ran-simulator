package handover

import (
	"strconv"
	"sync"

	"github.com/nfvri/ran-simulator/pkg/model"
)

// NewA3HandoverHandler returns A3HandoverHandler object
func NewA3HandoverHandler() *A3HandoverHandler {
	return &A3HandoverHandler{
		Chans: A3HandoverChannel{
			InputChan:  make(chan model.UE),
			OutputChan: make(chan A3HandoverDecision),
		},
	}
}

// A3HandoverHandler is A3 handover handler
type A3HandoverHandler struct {
	Chans        A3HandoverChannel
	HandlerMutex sync.RWMutex
	Model        *model.Model
}

// A3HandoverChannel struct has channels used in A3 handover handler
type A3HandoverChannel struct {
	InputChan  chan model.UE
	OutputChan chan A3HandoverDecision
}

// Run starts A3 handover handler
func (h *A3HandoverHandler) Run() {
	for ue := range h.Chans.InputChan {
		sCell := *h.Model.GetServingCells(ue.IMSI)[0]
		tCell := h.getTargetCell(ue)
		feasible := isHOFeasible(tCell, ue)
		h.Chans.OutputChan <- A3HandoverDecision{
			UE:          ue,
			ServingCell: sCell,
			TargetCell:  tCell,
			Feasible:    feasible,
		}
	}
}

func isHOFeasible(tCell model.Cell, ue model.UE) bool {
	return true
}

func (h *A3HandoverHandler) getTargetCell(ue model.UE) model.Cell {
	var targetCell model.Cell
	var bestRSRP float64
	flag := false

	for _, cscell := range ue.Cells {
		tmpRSRP := cscell.Rsrp
		if !flag {
			ncgiStr := strconv.FormatUint(uint64(cscell.NCGI), 10)
			targetCell = h.Model.Cells[ncgiStr]
			bestRSRP = tmpRSRP
			flag = true
			continue
		}

		if tmpRSRP > bestRSRP {
			ncgiStr := strconv.FormatUint(uint64(cscell.NCGI), 10)
			targetCell = h.Model.Cells[ncgiStr]
			bestRSRP = tmpRSRP
		}
	}
	return targetCell
}

// A3HandoverDecision struct has A3 handover decision information
type A3HandoverDecision struct {
	UE          model.UE
	ServingCell model.Cell
	TargetCell  model.Cell
	Feasible    bool
}
