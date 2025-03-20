// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package handover

import (
	"reflect"
	"strconv"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/signal"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	log "github.com/sirupsen/logrus"
)

var logHoCtrl = logging.GetLogger("handover", "controller")

// NewHOController returns the hanover controller
func NewHOController(hoType HOType, ho A3Handover) HOController {
	return &hoController{
		hoType:     hoType,
		inputChan:  make(chan model.UE),
		outputChan: make(chan HandoverDecision),
		HoHandler:  ho,
	}
}

// HOController is an abstraction of the handover controller
type HOController interface {
	// Start starts handover controller
	Start()

	// GetInputChan returns input channel
	GetInputChan() chan model.UE

	// GetOutputChan returns output channel
	GetOutputChan() chan HandoverDecision
}

// HandoverDecision struct has handover decision information
type HandoverDecision struct {
	UE              model.UE
	TargetCAScheme  bw.CAScheme
	TargetCellNcgis []types.NCGI
}

// HOType is the type of handover - currently it is string
type HOType string

const (
	A3 HOType = "A3"
)

type hoController struct {
	hoType     HOType
	inputChan  chan model.UE
	outputChan chan HandoverDecision
	HoHandler  A3Handover
}

func (h *hoController) Start() {
	switch h.hoType {
	case A3:
		h.startA3HandoverHandler()
	}
}

func (h *hoController) startA3HandoverHandler() {
	log.Info("Handover controller starting with A3HandoveHandler")
	go h.HoHandler.Start()
	// for input
	go h.forwardReportToA3HandoverHandler(h.HoHandler)
	//for output
	go h.forwardHandoverDecision(h.HoHandler)
}

func (h *hoController) forwardReportToA3HandoverHandler(handler A3Handover) {
	for ue := range h.inputChan {
		log.Debugf("[input] Measurement report for HO decision: %v", ue)
		handler.PushMeasurementEventA3(ue)
	}
}

func (h *hoController) forwardHandoverDecision(handler A3Handover) {
	for hoDecision := range handler.GetOutputChan() {
		log.Debugf("[output] Handover decision: %v", hoDecision)
		h.outputChan <- hoDecision
	}
}

func (h *hoController) GetInputChan() chan model.UE {
	return h.inputChan
}

func (h *hoController) GetOutputChan() chan HandoverDecision {
	return h.outputChan
}

type HOExecutor interface {
	Execute(hoDecision HandoverDecision)
}

type DefaultHOExecutor struct {
	Model *model.Model
}

func (e *DefaultHOExecutor) Execute(hoDecision HandoverDecision) {

	log.Infof(
		`
	------------------------------------------------------------
	ue: %v | HANDOVER EXECUTION: tcells: %v 
	------------------------------------------------------------`,
		hoDecision.UE.IMSI,
		hoDecision.TargetCellNcgis,
	)

	isHandover := hoDecision.UE.RrcState == e2sm_mho.Rrcstatus_RRCSTATUS_CONNECTED
	if !isHandover {
		log.Warnf("ue: %v | not in state RRC_CONNECTED cannot perform handover", hoDecision.UE.IMSI)
		return
	}

	imsiStr := strconv.FormatUint(uint64(hoDecision.UE.IMSI), 10)
	ue := e.Model.UEs[imsiStr]

	servCells := e.Model.GetServingCells(ue.IMSI)
	servCellNCGIs := []types.NCGI{}
	for c := range servCells {
		servCellNCGIs = append(servCellNCGIs, servCells[c].NCGI)
	}

	noTargetCells := len(hoDecision.TargetCellNcgis) == 0
	if noTargetCells {
		log.Warnf("ue: %v | no target cells found", ue.IMSI)
		bw.ReleaseBW(servCells, ue)
		ue.RrcState = e2sm_mho.Rrcstatus_RRCSTATUS_IDLE
		e.Model.UpdateServiceMappings(ue.IMSI, servCellNCGIs, hoDecision.TargetCellNcgis)
		return
	}

	noChange := reflect.DeepEqual(servCellNCGIs, hoDecision.TargetCellNcgis)
	if noChange {
		log.Infof("ue: %v | serving cells did not change skipping HO", ue.IMSI)
		return
	}

	// lock on all serving cells that will stop serving the ue
	// so as to release bandwidth
	stoppedServingCells := e.getStoppedServingCells(servCells, hoDecision)
	for c := range stoppedServingCells {
		stoppedServingCell := stoppedServingCells[c]
		servCellNCGIs = append(servCellNCGIs, stoppedServingCell.NCGI)
		stoppedServingCell.Lock()
		defer stoppedServingCell.Unlock()
	}
	// lock on all target cells not already serving
	targetCells := e.getTargetCells(hoDecision, ue)
	for c := range targetCells {
		targetCell := targetCells[c]
		targetCell.Lock()
		defer targetCell.Unlock()
	}
	// lock service mappings
	e.Model.ServiceMappings.Lock()
	defer e.Model.ServiceMappings.Unlock()

	releasedBwps := bw.ReleaseBW(stoppedServingCells, ue)
	log.Infof("ue: %v | UpdateServiceMappings", ue.IMSI)
	e.Model.UpdateServiceMappings(ue.IMSI, servCellNCGIs, hoDecision.TargetCellNcgis)
	log.Infof("ue: %v | ComputeCellMetricsFor", ue.IMSI)
	e.ComputeCellMetricsFor(ue)
	log.Infof("ue: %v | AllocateBandwidth", ue.IMSI)
	bw.AllocateBandwidth(ue, releasedBwps, stoppedServingCells, targetCells, hoDecision.TargetCAScheme, e.Model.GetServedUEs)
	log.Infof("ue: %v | logHO", ue.IMSI)
	logHO(hoDecision, servCellNCGIs)

}

func (e *DefaultHOExecutor) getTargetCells(hoDecision HandoverDecision, ue *model.UE) []*model.Cell {
	targetCells := []*model.Cell{}
	for _, ncgi := range hoDecision.TargetCellNcgis {
		targetCellAlreadyServing := false
		for _, ueServingCell := range ue.ServingCells {
			if ueServingCell.NCGI == ncgi {
				targetCellAlreadyServing = true
				break
			}
		}
		if targetCellAlreadyServing {
			continue
		}
		targetCellNcgiStr := strconv.FormatUint(uint64(ncgi), 10)
		targetCell := e.Model.Cells[targetCellNcgiStr]
		targetCells = append(targetCells, targetCell)
	}
	return targetCells
}

func (*DefaultHOExecutor) getStoppedServingCells(servCells []*model.Cell, hoDecision HandoverDecision) []*model.Cell {
	stoppedServingCells := []*model.Cell{}
	for servCellIndex := range servCells {
		servCell := servCells[servCellIndex]
		targetCellsContainServing := false
		for _, tCellNCGI := range hoDecision.TargetCellNcgis {
			if tCellNCGI == servCell.NCGI {
				targetCellsContainServing = true
				break
			}
		}
		if targetCellsContainServing {
			continue
		}
		stoppedServingCells = append(stoppedServingCells, servCell)
	}
	return stoppedServingCells
}

func logHO(hoDecision HandoverDecision, sourceCellNCGIs []types.NCGI) {
	log.Debug("Handover COMPLETE")
	log.Debugf("len(targetCellNcgis): %v", len(hoDecision.TargetCellNcgis))
	log.Debugf("len(sourceCellNCGIs): %v", len(sourceCellNCGIs))
	log.Debug("==================================================================")
	log.Debugf("HO is done successfully: %v to %v", hoDecision.UE.IMSI, hoDecision.TargetCellNcgis)
}

// ComputeCellMetricsFor recomputes the signal metrics for the serving and neighbor cells of the ue.
func (e *DefaultHOExecutor) ComputeCellMetricsFor(ue *model.UE) {

	for c := range ue.ServingCells {
		sCell := e.Model.Cells[strconv.FormatUint(uint64(ue.ServingCells[c].NCGI), 10)]
		servCell := ue.ServingCells[c]
		servCell.Rsrp = signal.RSRP(ue, sCell, servCell.BeamID)
		iBeamIDs, interferingCells := signal.GetInterferingBeams(ue.Location, sCell, servCell.BeamID, e.Model.Cells)
		servCell.Sinr = signal.Sinr(ue.Location, ue.Height, sCell, servCell.BeamID, iBeamIDs, interferingCells)
		servCell.Rsrq = signal.RSRQ(servCell.Sinr, servCell.AvailPrbsDl)
	}

	if len(ue.ServingCells) > 0 {
		ue.FiveQi = signal.GetCQI(ue.ServingCells[0].Sinr)
	} else {
		ue.FiveQi = 1
	}

	for c := range ue.NeighborCells {
		neighCell := ue.NeighborCells[c]
		nCell := e.Model.Cells[strconv.FormatUint(uint64(neighCell.NCGI), 10)]
		neighCell.Rsrp = signal.RSRP(ue, nCell, neighCell.BeamID)
		interfBeamIDs, interfCells := signal.GetInterferingBeams(ue.Location, nCell, neighCell.BeamID, e.Model.Cells)
		neighCell.Sinr = signal.Sinr(ue.Location, ue.Height, nCell, neighCell.BeamID, interfBeamIDs, interfCells)
		neighCell.Rsrq = signal.RSRQ(neighCell.Sinr, neighCell.AvailPrbsDl)
	}

	ueCopy := *ue
	e.Model.UpsertUE(&ueCopy)
}
