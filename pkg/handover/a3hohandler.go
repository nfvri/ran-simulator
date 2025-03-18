package handover

import (
	"strconv"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/signal"
	"github.com/sirupsen/logrus"
)

// NewA3HandoverHandler returns A3HandoverHandler object
func NewA3HandoverHandler(m *model.Model) *A3HandoverHandler {
	return &A3HandoverHandler{
		Chans: A3HandoverChannel{
			InputChan:  make(chan model.UE),
			OutputChan: make(chan HandoverDecision),
		},
		cas: map[model.ConnectivityType]bw.CarrierAggregator{
			model.EUTRA: bw.NewCarrierAggregatorEUTRA(),
			model.NR:    bw.NewCarrierAggregatorNR(),
		},
		model: m,
	}
}

// A3HandoverHandler is A3 handover handler
type A3HandoverHandler struct {
	Chans A3HandoverChannel
	cas   map[model.ConnectivityType]bw.CarrierAggregator
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
		ueCells := []*model.UECell{}
		ueCells = append(ueCells, ue.ServingCells...)
		ueCells = append(ueCells, ue.NeighborCells...)
		topKByRSRP := signal.TopKCellsByRSRP(ue, ueCells, len(ue.ServingCells))
		topKNcgis := []types.NCGI{}
		for _, ueCell := range topKByRSRP {
			topKNcgis = append(topKNcgis, ueCell.NCGI)
		}
		targetCellNCGIs, caScheme := h.selectTargetCells(ue, topKNcgis)
		h.Chans.OutputChan <- HandoverDecision{
			UE:              ue,
			TargetCAScheme:  caScheme,
			TargetCellNcgis: targetCellNCGIs,
		}
	}
}

func (h *A3HandoverHandler) selectTargetCells(ue model.UE, rankedNCGIs []types.NCGI) ([]types.NCGI, bw.CAScheme) {

	logrus.Infof(`
	------------------------------------------------------------
	ue: %v | HANDOVER PREPARATION
	------------------------------------------------------------
	`, ue.IMSI)

	model.LogUECells(&ue)
	logrus.Infof("ue: %v | selecting target cells", ue.IMSI)

	if len(rankedNCGIs) == 0 {
		return rankedNCGIs, bw.CAScheme{}
	}

	var maxChannelBwDL uint32 = 0
	for _, ueSCell := range ue.ServingCells {
		ncgiStr := strconv.FormatUint(uint64(ueSCell.NCGI), 10)
		cell := h.model.Cells[ncgiStr]
		for c := range cell.Carriers {
			if maxChannelBwDL < cell.Carriers[c].BsChannelBwDL {
				maxChannelBwDL = cell.Carriers[c].BsChannelBwDL
			}
		}
	}

	ueSupportsCA := len(ue.SupportedBandCombinations) > 0
	if !ueSupportsCA {
		logrus.Warnf("ue: %v | does not support CA", ue.IMSI)
		return []types.NCGI{rankedNCGIs[0]}, bw.CAScheme{}
	}

	logrus.Infof("ue: %v | supports CA!", ue.IMSI)

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

	logrus.Infof("ue: %v | attempting selfSchedulingCells: %+v", ue.IMSI, selfSchedulingCells)

	// self scheduling
	ueRequiredPRBsDL, ueRequiredPRBsUL := bw.CurrPRBsUsed(&ue)
	for c := range selfSchedulingCells {
		cell := selfSchedulingCells[c]
		servedUEs := h.model.GetServedUEs(cell.NCGI)
		cellAvailPrbsUL, cellAvailPrbsDL, err := bw.GetCellAvailPRBs(cell, servedUEs, &ue)
		if err != nil {
			logrus.Errorf("ue: %v | failed to get available prbs for cell: %v, %v", ue.IMSI, cell.NCGI, err)
			continue
		}
		logrus.Infof(
			`ue: %v |
			ueRequiredPRBsUL:%v vs cellAvailPrbsUL:%v,
			ueRequiredPRBsDL:%v vs cellAvailPrbsDL:%v`,
			ue.IMSI,
			ueRequiredPRBsUL, cellAvailPrbsUL,
			ueRequiredPRBsDL, cellAvailPrbsDL,
		)

		if cellAvailPrbsUL >= ueRequiredPRBsUL && cellAvailPrbsDL >= ueRequiredPRBsDL {
			logrus.Infof("found selfSchedulingCell: %v", cell.NCGI)
			return []types.NCGI{cell.NCGI}, bw.CAScheme{}
		}
	}

	// Check Bands for CA
	logrus.Infof("ue: %v | attempting ccSchedulingCells: %+v", ue.IMSI, ccSchedulingCells)
	supportedCACombos, cellsByEUTRABand, cellsByNRBand := bw.FindSupportedCACombos(&ue, h.cas, ccSchedulingCells)

	anyValidBandCombo := false
	for ct := range supportedCACombos {
		anyValidBandCombo = len(supportedCACombos[ct]) > 0
		if anyValidBandCombo {
			break
		}
	}

	if !anyValidBandCombo {
		logrus.Warnf("ue: %v | failed to find supported band combinations", ue.IMSI)
		return []types.NCGI{rankedNCGIs[0]}, bw.CAScheme{}
	}

	logrus.Infof("ue: %v | validCACombinations: %+v", ue.IMSI, supportedCACombos)

	// Check available BW for CA
	feasibleCASchemes := bw.GetFeasibleCASchemes(supportedCACombos, cellsByEUTRABand, cellsByNRBand, h.model, &ue)
	logrus.Infof("ue: %v | feasibleCASchemes: %+v", ue.IMSI, feasibleCASchemes)
	anyFeasibleCAScheme := len(feasibleCASchemes) > 0

	if !anyFeasibleCAScheme {
		logrus.Warnf("ue: %v | failed to find feasible CA scheme", ue.IMSI)
		logrus.Info("\n========================================================================\n")
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
	logrus.Info("\n========================================================================\n")

	return targetCellNcgis, selectedCAScheme
}
