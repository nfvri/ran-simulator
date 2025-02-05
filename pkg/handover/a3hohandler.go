package handover

import (
	"math"
	"sort"
	"strconv"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/sirupsen/logrus"
)

const MIN_ACCEPTABLE_RSRP = -110.0

type CarrierAggregator interface {
	GetValidCACombinations(cells []*model.Cell, supportedBandsInfo map[model.ConnectivityType][]*model.BandSupportInfo) (validCABandCombos [][]string, cellsByBand map[string][]*model.Cell)
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
		targetCellNCGIs, caScheme := h.selectTargetCells(ue, h.rankTargetCellsByRSRP(ue))
		h.Chans.OutputChan <- HandoverDecision{
			UE:              ue,
			TargetCAScheme:  caScheme,
			TargetCellNcgis: targetCellNCGIs,
		}
	}
}

func (h *A3HandoverHandler) rankTargetCellsByRSRP(ue model.UE) []types.NCGI {
	logrus.Info("[rankTargetCellsByRSRP]...")
	bestRSRPsByNCGI := map[types.NCGI]float64{}

	for _, ueSCell := range ue.ServingCells {
		bestRSRPsByNCGI[ueSCell.NCGI] = ueSCell.Rsrp
	}

	for _, cscell := range ue.NeighborCells {
		var ncgiToReplace types.NCGI = 0.0
		minRSRP := math.Inf(1)
		for ncgi, rsrp := range bestRSRPsByNCGI {
			if minRSRP > rsrp {
				minRSRP = rsrp
				ncgiToReplace = ncgi
			}
		}

		if ncgiToReplace != 0.0 && minRSRP < cscell.Rsrp {
			delete(bestRSRPsByNCGI, ncgiToReplace)
			bestRSRPsByNCGI[cscell.NCGI] = cscell.Rsrp
		}
	}

	rankedNCGIs := []types.NCGI{}
	for ncgi := range bestRSRPsByNCGI {
		if bestRSRPsByNCGI[ncgi] > MIN_ACCEPTABLE_RSRP {
			rankedNCGIs = append(rankedNCGIs, ncgi)
		}
	}

	sort.Slice(rankedNCGIs, func(i, j int) bool {
		return bestRSRPsByNCGI[rankedNCGIs[i]] > bestRSRPsByNCGI[rankedNCGIs[j]]
	})

	return rankedNCGIs
}

func (h *A3HandoverHandler) selectTargetCells(ue model.UE, rankedNCGIs []types.NCGI) ([]types.NCGI, bw.CAScheme) {

	logrus.Info("[selectTargetCells]...")

	if len(rankedNCGIs) == 0 {
		return rankedNCGIs, bw.CAScheme{}
	}

	var maxChannelBwDL uint32 = 0
	for _, ueSCell := range ue.ServingCells {
		ncgiStr := strconv.FormatUint(uint64(ueSCell.NCGI), 10)
		cell := h.model.Cells[ncgiStr]
		if maxChannelBwDL < cell.Channel.BsChannelBwDL {
			maxChannelBwDL = cell.Channel.BsChannelBwDL
		}
	}

	ueSupportsCA := len(ue.SupportedBandCombinations) > 0
	if !ueSupportsCA {
		logrus.Infof("UE %v does not support CA", ue.IMSI)
		return []types.NCGI{rankedNCGIs[0]}, bw.CAScheme{}
	}

	logrus.Infof("ue %v supports CA! ", ue.IMSI)

	ccSchedulingCells := []*model.Cell{}
	selfSchedulingCells := []*model.Cell{}
	for _, ncgi := range rankedNCGIs {
		ncgiStr := strconv.FormatUint(uint64(ncgi), 10)
		cell := h.model.Cells[ncgiStr]
		crossCarrierSchedulingSupported := cell.SchedulingCellInfo == model.SCHEDULING_CELL_INFO_OTHER
		if crossCarrierSchedulingSupported {
			ccSchedulingCells = append(ccSchedulingCells, cell)
		} else {
			selfSchedulingCells = append(selfSchedulingCells, cell)
		}
	}

	logrus.Infof("attempting selfSchedulingCells: %+v", selfSchedulingCells)

	ueRequiredPRBsDL, ueRequiredPRBsUL := bw.CurrPRBsUsed(&ue)
	for c := range selfSchedulingCells {
		cell := selfSchedulingCells[c]
		servedUEs := h.model.GetServedUEs(cell.NCGI)
		cellAvailPrbsUL, cellAvailPrbsDL, err := bw.GetCellAvailPRBs(cell, servedUEs, &ue)
		logrus.Infof(
			`ue:%v, 
			ueRequiredPRBsUL:%v vs cellAvailPrbsUL:%v, 
			ueRequiredPRBsDL:%v vs cellAvailPrbsDL:%v`,
			ue.IMSI,
			ueRequiredPRBsUL, cellAvailPrbsUL,
			ueRequiredPRBsDL, cellAvailPrbsDL,
		)

		if err != nil {
			continue
		}
		if cellAvailPrbsUL >= ueRequiredPRBsUL && cellAvailPrbsDL >= ueRequiredPRBsDL {
			logrus.Infof("found selfSchedulingCell: %v", cell.NCGI)
			return []types.NCGI{cell.NCGI}, bw.CAScheme{}
		}
	}

	logrus.Infof("attempting ccSchedulingCells: %+v", ccSchedulingCells)

	validCACombinations, cellsByBand := h.ca.GetValidCACombinations(ccSchedulingCells, ue.SupportedBandCombinations)

	logrus.Infof("validCACombinations: %+v", validCACombinations)
	feasibleCASchemes := bw.GetFeasibleCASchemes(validCACombinations, cellsByBand, h.model, &ue)
	logrus.Infof("feasibleCASchemes: %+v", feasibleCASchemes)
	anyFeasibleCAScheme := len(feasibleCASchemes) > 0

	if !anyFeasibleCAScheme {
		return rankedNCGIs, bw.CAScheme{}
	}

	// naively select first feasible
	selectedCAScheme := feasibleCASchemes[0]
	targetCellNcgis := []types.NCGI{}
	for _, c := range selectedCAScheme.FixedAllocCells {
		targetCellNcgis = append(targetCellNcgis, c.NCGI)
	}
	for _, c := range selectedCAScheme.ReallocCells {
		targetCellNcgis = append(targetCellNcgis, c.NCGI)
	}

	return targetCellNcgis, selectedCAScheme
}
