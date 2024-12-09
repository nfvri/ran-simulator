package handover

import (
	"sort"

	"github.com/nfvri/ran-simulator/pkg/model"
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

		uePCellNcgi := ue.ServingCells[0].NCGI
		tCellNcgis := h.getTargetServingCells(ue)
		h.Chans.OutputChan <- HandoverDecision{
			UE:              ue,
			SourceCellNcgi:  uePCellNcgi,
			TargetCellNcgis: tCellNcgis,
		}
	}
}

func (h *A3HandoverHandler) getTargetServingCells(ue model.UE) []types.NCGI {
	bestRSRPs := map[types.NCGI]float64{}
	for _, ueSCell := range ue.ServingCells {
		if ueSCell.Rsrp > -110 {
			bestRSRPs[ueSCell.NCGI] = ueSCell.Rsrp
		}
	}

	for _, cscell := range ue.NeighborCells {
		var replacedNCGI types.NCGI = 0.0
		for ncgi, rsrp := range bestRSRPs {
			if cscell.Rsrp > rsrp {
				replacedNCGI = ncgi
				break
			}
		}
		if replacedNCGI != 0.0 {
			delete(bestRSRPs, replacedNCGI)
		}
	}

	newServingCellNCGIs := []types.NCGI{}
	for ncgi := range bestRSRPs {
		newServingCellNCGIs = append(newServingCellNCGIs, ncgi)
	}

	sort.Slice(newServingCellNCGIs, func(i, j int) bool {
		return bestRSRPs[newServingCellNCGIs[i]] > bestRSRPs[newServingCellNCGIs[j]]
	})

	return newServingCellNCGIs
}
