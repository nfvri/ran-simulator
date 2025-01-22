package handover

import (
	"sort"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/sirupsen/logrus"
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

func (h *A3HandoverHandler) getTargetCells(ue *model.UE, bestNCGIsByRSRP []types.NCGI) []*model.Cell {
	var maxChannelBwDL uint32 = 0
	targetCells := []*model.Cell{}
	for cellIndex := range h.model.Cells {
		cell := h.model.Cells[cellIndex]
		servCell := ue.GetServingCell(cell.NCGI)
		if servCell != nil && maxChannelBwDL < cell.Channel.BsChannelBwDL {
			maxChannelBwDL = cell.Channel.BsChannelBwDL
		}
		for _, ncgi := range bestNCGIsByRSRP {
			crossCarrierSchedulingSupported := cell.SchedulingCellInfo == model.SCHEDULING_CELL_INFO_OTHER
			if cell.NCGI == ncgi && crossCarrierSchedulingSupported {
				targetCells = append(targetCells, cell)
			}
		}
	}

	bwc, bwcFound := bw.GetBandwidthClassNR(ue, maxChannelBwDL)
	caSupported := len(targetCells) > 1 && bwcFound && bwc.NumContiguousCC >= 2
	if caSupported {
		validCACombinations, cellsByBand := h.ca.GetValidCACombinations(targetCells)
		logrus.Infof("\n=================\n%+v, \n%+v \n=================\n", validCACombinations, cellsByBand)
	}
	return targetCells
}

func (h *A3HandoverHandler) hasSufficientPRBS(
	validCABandCombos [][]string,
	cellsByBand map[string][]*model.Cell,
	isFR2 bool,
	ue *model.UE) bool {

	requiredPRBsDL, requiredPRBsUL := bw.CurrPRBsUsed(ue)

	// Iterate over CA band combinations
	for _, bandCombo := range validCABandCombos {
		// Extract cells for the band combination
		var caScheme []*model.Cell
		for _, band := range bandCombo {
			if cells, exists := cellsByBand[band]; exists {
				caScheme = append(caScheme, cells...)
			}
		}

		// Check PRB availability across the gathered cells
		totalAvailableBWDL := 0.0
		totalAvailableBWUL := 0.0

		for _, cell := range caScheme {
			// Get served UEs to calculate current bandwidth usage in the cell
			servedUEs := h.model.GetServedUEs(cell.NCGI)

			usedBWDL := 0.0
			usedBWUL := 0.0
			for _, servedUE := range servedUEs {
				for _, bwp := range servedUE.GetServingCell(cell.NCGI).BwpRefs {
					if bwp.Downlink {
						usedBWDL += float64(bwp.NumberOfRBs) * float64(bwp.Scs) * 12
					} else {
						usedBWUL += float64(bwp.NumberOfRBs) * float64(bwp.Scs) * 12
					}
				}
			}

			// Available bandwidth in DL and UL
			totalAvailableBWDL += float64(cell.Channel.BsChannelBwDL) - usedBWDL
			totalAvailableBWUL += float64(cell.Channel.BsChannelBwUL) - usedBWUL
		}

		// Determine minimum SCS
		var minSCS int
		if isFR2 {
			minSCS = bw.FRtoSCS["FR2"][0]
		} else {
			minSCS = bw.FRtoSCS["FR1"][0]
		}

		// Calculate PRBs based on available bandwidth and minimum SCS
		DLPRBS, _ := bw.GetPRBs(totalAvailableBWDL, minSCS, isFR2)
		ULPRBS, _ := bw.GetPRBs(totalAvailableBWUL, minSCS, isFR2)

		// Check if PRB requirements are met
		if (ULPRBS > requiredPRBsUL) && (DLPRBS > requiredPRBsDL) {
			return true
		}
	}

	// If no combination meets the criteria, return false
	return false
}
