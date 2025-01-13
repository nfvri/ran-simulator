package handover

import (
	"sort"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
	"github.com/nfvri/ran-simulator/pkg/model"
)

const MIN_ACCEPTABLE_RSRP = -110.0

// NewA3HandoverHandler returns A3HandoverHandler object
func NewA3HandoverHandler(ca *bw.CarrierAggregatorNR, m *model.Model) *A3HandoverHandler {
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
	ca    *bw.CarrierAggregatorNR
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
			TargetCellNcgis: h.rankTargetCellsByRSRP(ue),
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

func (h *A3HandoverHandler) getValidCACombinations(targetCellNCGIs []types.NCGI) {
	// TODO: search neighbors for allowed CA combinations
	// and try k=2,3,4...6 to find a CA that covers bw requirements
	// PRBS, SCS increasing in freq
	targetCellsBandsDL := []string{}
	// targetCellsBandsUL := []string{}
	for cellIndex := range h.model.Cells {
		cell := h.model.Cells[cellIndex]
		for _, ncgi := range targetCellNCGIs {
			if cell.NCGI == ncgi {
				// TODO: check both DL, UL
				targetCellsBandsDL = append(targetCellsBandsDL, bw.GetBandName(cell.ArfcnDL, bw.DL))
				// [n1, n38, n5, n3]
				// n1, n3, n5, n38
				// n1_n3
				// n1_n3_n5
				// n1_n3_n5_n38 -> c1, c4, c8
				// n2_n7 -> c1, c4, c8

			}
		}
	}
	// TODO: fix sorting to sort on band number
	sort.Strings(targetCellsBandsDL)
	// TODO: concatenate and check if valid combinations using bfs
	// h.ca.IsValidBandCombination()

}

func (h *A3HandoverHandler) hasSufficientPRBS(caScheme [][]*model.Cell) {
	// uePRBsUsed := bw.CurrPRBsUsed(&ue)

	// caSchemes := [][]*model.Cell{
	// 	[]*model.Cell{c1, c2....ck}, Combo1
	// 	[]*model.Cell{c1....cl}, Combo2
	// }

	// c := &model.Cell{}
	// h.model.GetServedUEs(c.NCGI)
	// c.Channel.BsChannelBwDL - usedBWDL -> availableBW (MHz)
	// c.Channel.BsChannelBwUL - usedBWUL-> availableBW (MHz)
	// bw.GetPRBs()
}
