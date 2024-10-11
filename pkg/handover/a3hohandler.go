package handover

import (
	"math"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
)

// NewA3HandoverHandler returns A3HandoverHandler object
func NewA3HandoverHandler() *A3HandoverHandler {
	return &A3HandoverHandler{
		Chans: A3HandoverChannel{
			InputChan:  make(chan model.UE),
			OutputChan: make(chan HandoverDecision),
		},
	}
}

// A3HandoverHandler is A3 handover handler
type A3HandoverHandler struct {
	Chans A3HandoverChannel
}

// A3HandoverChannel struct has channels used in A3 handover handler
type A3HandoverChannel struct {
	InputChan  chan model.UE
	OutputChan chan HandoverDecision
}

// Run starts A3 handover handler
func (h *A3HandoverHandler) Run() {
	for ue := range h.Chans.InputChan {
		sourceCellNcgi := ue.Cell.NCGI
		tCellNcgi := h.getTargetCell(ue)
		h.Chans.OutputChan <- HandoverDecision{
			UE:             ue,
			SourceCellNcgi: sourceCellNcgi,
			TargetCellNcgi: tCellNcgi,
		}
	}
}

func (h *A3HandoverHandler) getTargetCell(ue model.UE) types.NCGI {
	targetCellNcgi := ue.Cell.NCGI
	bestRSRP := ue.Cell.Rsrp

	for _, cscell := range ue.Cells {
		if cscell.Rsrp > bestRSRP {
			targetCellNcgi = cscell.NCGI
			bestRSRP = cscell.Rsrp
		}
	}

	return utils.If(bestRSRP == math.Inf(-1), 0, targetCellNcgi)
}
