package handover

import (
	"sort"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
	"github.com/nfvri/ran-simulator/pkg/model"
)

const MIN_ACCEPTABLE_RSRP = -110.0

type CarrierAggregator interface {
	GetValidCACombinations(cells []*model.Cell) (validCABandCombos [][]string, cellsByBand map[string][]*model.Cell)
}

// NewA3HandoverHandler returns A3HandoverHandler object
func NewA3HandoverHandler(ca CarrierAggregator, m *model.Model) *A3HandoverHandler {
	return &A3HandoverHandler{
		Chans: A3HandoverChannel{
			InputChan:  make(chan model.UE),
			OutputChan: make(chan HandoverDecision),
		},
		ca:    ca,
		model: m,
	}
}

// A3HandoverHandler is A3 handover handler
type A3HandoverHandler struct {
	Chans A3HandoverChannel
	ca    CarrierAggregator
	model *model.Model
}

// A3HandoverChannel struct has channels used in A3 handover handler
type A3HandoverChannel struct {
	InputChan  chan model.UE
	OutputChan chan HandoverDecision
}

// Run starts A3 handover handler
func (h *A3HandoverHandler) Run() {
	for ue := range h.Chans.InputChan {
		h.Chans.OutputChan <- HandoverDecision{
			UE:              ue,
			TargetCellNcgis: h.getTargetCells(ue, h.rankTargetCellsByRSRP(ue)),
		}
	}
}

func (h *A3HandoverHandler) rankTargetCellsByRSRP(ue model.UE) []types.NCGI {
	bestRSRPs := map[types.NCGI]float64{}

	for _, ueSCell := range ue.ServingCells {
		if ueSCell.Rsrp > MIN_ACCEPTABLE_RSRP {
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

	bestNCGIsByRSRP := []types.NCGI{}
	for ncgi := range bestRSRPs {
		bestNCGIsByRSRP = append(bestNCGIsByRSRP, ncgi)
	}

	sort.Slice(bestNCGIsByRSRP, func(i, j int) bool {
		return bestRSRPs[bestNCGIsByRSRP[i]] > bestRSRPs[bestNCGIsByRSRP[j]]
	})

	return bestNCGIsByRSRP
}

func (h *A3HandoverHandler) getTargetCells(ue model.UE, bestNCGIsByRSRP []types.NCGI) []types.NCGI {

	var maxChannelBwDL uint32 = 0
	ueBWC, bwcFound := bw.GetBandwidthClassNR(&ue, maxChannelBwDL)
	ueSupportsCA := bwcFound && ueBWC.NumContiguousCC >= 2

	if !ueSupportsCA {
		return bestNCGIsByRSRP
	}

	targetCells := []*model.Cell{}
	for cellIndex := range h.model.Cells {
		cell := h.model.Cells[cellIndex]
		_, isServingCell := ue.GetServingCell(cell.NCGI)
		if isServingCell && maxChannelBwDL < cell.Channel.BsChannelBwDL {
			maxChannelBwDL = cell.Channel.BsChannelBwDL
		}
		for _, ncgi := range bestNCGIsByRSRP {
			crossCarrierSchedulingSupported := cell.SchedulingCellInfo == model.SCHEDULING_CELL_INFO_OTHER
			if cell.NCGI == ncgi && crossCarrierSchedulingSupported {
				targetCells = append(targetCells, cell)
			}
		}
	}

	validCACombinations, cellsByBand := h.ca.GetValidCACombinations(targetCells)
	feasibleCASchemes := bw.GetFeasibleCASchemes(validCACombinations, cellsByBand, h.model, &ue)

	targetCellNcgis := []types.NCGI{}
	for _, c := range feasibleCASchemes[0].Cells {
		targetCellNcgis = append(targetCellNcgis, c.NCGI)
	}

	return targetCellNcgis
}
