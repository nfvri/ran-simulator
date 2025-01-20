package handover

import (
	"sort"
	"strconv"
	"strings"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
	"k8s.io/utils/strings/slices"
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

func (h *A3HandoverHandler) GetValidCACombinations(targetCellNCGIs []types.NCGI) (validCABandCombos [][]string, cellsByBand map[bw.BandNR][]*model.Cell) {
	validCABandCombos = make([][]string, 0)
	cellsByBand = make(map[bw.BandNR][]*model.Cell)
	for cellIndex := range h.model.Cells {
		cell := h.model.Cells[cellIndex]
		for _, ncgi := range targetCellNCGIs {
			crossCarrierSchedulingSupported := cell.SchedulingCellInfo == model.SCHEDULING_CELL_INFO_OTHER
			if cell.NCGI == ncgi && crossCarrierSchedulingSupported {
				// FIXME: For cells on different nodes
				// we should consult DUAL CONNECTIVITY combinations in:
				// https://www.etsi.org/deliver/etsi_ts/138100_138199/13810103/15.02.00_60/ts_13810103v150200p.pdf
				// https://www.sqimway.com/nr_nrdc.php
				// https://www.sqimway.com/nr_endc.php
				// https://www.sqimway.com/nr_nedc.php
				arfcn := utils.If(cell.ArfcnDL > 0, cell.ArfcnDL, cell.ArfcnUL)
				cellBand, found := bw.GetBand(arfcn, bw.DL)
				if !found {
					continue
				}
				cellsByBand[cellBand] = append(cellsByBand[cellBand], cell)
			}
		}
	}

	targetCellBands := []bw.BandNR{}
	for b := range cellsByBand {
		targetCellBands = append(targetCellBands, b)
	}

	caBandCombos := getCABandCombinations(sortNRBands(targetCellBands))

	for _, caBandCombo := range caBandCombos {
		if h.ca.IsValidBandCombination(caBandCombo) {
			validCABandCombos = append(validCABandCombos, strings.Split(caBandCombo, "_"))
		}
	}

	return

}

// Function to sort the slice of strings
func sortNRBands(bands []bw.BandNR) []bw.BandNR {
	bandNumber := func(s string) int {
		for _, char := range s {
			if char >= '0' && char <= '9' {
				num, _ := strconv.Atoi(s[1:])
				return num
			}
		}
		return 0
	}
	sort.Slice(bands, func(i, j int) bool {
		return bandNumber(bands[i].Name) < bandNumber(bands[j].Name)
	})
	return bands
}

func getCABandCombinations(sortedBandsNR []bw.BandNR) []string {
	var combinations []string
	queue := []string{}

	for i := 0; i < len(sortedBandsNR); i++ {
		for j := i + 1; j < len(sortedBandsNR); j++ {
			initialCombo := sortedBandsNR[i].Name + "_" + sortedBandsNR[j].Name
			queue = append(queue, initialCombo)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		combinations = append(combinations, current)

		for _, b := range sortedBandsNR {
			if !strings.Contains(current, b.Name) {
				newCombination := current + "_" + b.Name
				if !slices.Contains(combinations, newCombination) {
					queue = append(queue, newCombination)
				}
			}
		}
	}

	return combinations
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
