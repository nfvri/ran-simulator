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
	"github.com/nfvri/ran-simulator/pkg/utils"
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

	log.Debug("---------------------------------- ")
	log.Debugf("handover:  ue: %v [pcell: %v ==> tcell: %v]",
		hoDecision.UE.IMSI,
		hoDecision.TargetCellNcgis,
	)
	log.Debug("---------------------------------- ")

	isHandover := hoDecision.UE.RrcState == e2sm_mho.Rrcstatus_RRCSTATUS_CONNECTED
	if !isHandover {
		return
	}

	imsiStr := strconv.FormatUint(uint64(hoDecision.UE.IMSI), 10)
	ue := e.Model.UEs[imsiStr]

	// lock on all serving cells not in target cells
	servCells := e.Model.GetServingCells(ue.IMSI)
	servCellNCGIs := []types.NCGI{}
	for servCellIndex := range servCells {
		servCell := servCells[servCellIndex]
		servCellNCGIs = append(servCellNCGIs, servCell.NCGI)
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
		servCell.Lock()
		defer servCell.Unlock()
	}

	noTargetCells := len(hoDecision.TargetCellNcgis) == 0
	if noTargetCells {
		bw.ReleaseBWPs(servCells, ue)
		ue.RrcState = e2sm_mho.Rrcstatus_RRCSTATUS_IDLE
		e.Model.UpdateServiceMappings(ue.IMSI, servCellNCGIs, hoDecision.TargetCellNcgis)
		return
	}

	noChange := reflect.DeepEqual(servCellNCGIs, hoDecision.TargetCellNcgis)
	if noChange {
		return
	}

	// lock service mappings
	e.Model.ServiceMappings.Lock()
	defer e.Model.ServiceMappings.Unlock()

	// lock on all target cells not already serving
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
		targetCell.Lock()
		defer targetCell.Unlock()
	}

	// TODO: Check if target contains existing serving cells
	// if so exclude them from reallocation
	// first reallocate and then release bw?
	if len(hoDecision.TargetCellNcgis) != 0 {
		requestedBwps := bw.ReleaseBWPs(servCells, ue)
		e.Model.UpdateServiceMappings(ue.IMSI, servCellNCGIs, hoDecision.TargetCellNcgis)
		e.ComputeCellMetricsFor(ue)
		bw.ReallocateBW(ue, requestedBwps, targetCells, e.Model.GetServedUEs)
		logHO(hoDecision, servCellNCGIs)
	} else {
		ue.ServingCells[0].BwpRefs = []*model.Bwp{}
		ue.RrcState = e2sm_mho.Rrcstatus_RRCSTATUS_IDLE
		e.Model.UpdateServiceMappings(ue.IMSI, servCellNCGIs, hoDecision.TargetCellNcgis)
		return
	}

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

	uePCell := ue.ServingCells[0]
	sCell := e.Model.Cells[strconv.FormatUint(uint64(uePCell.NCGI), 10)]
	uePCell.Rsrp = signal.RSRP(ue, sCell)
	uePCell.Sinr = signal.Sinr(ue.Location, ue.Height, sCell, utils.GetNeighborCells(sCell, e.Model.Cells))
	uePCell.Rsrq = signal.RSRQ(uePCell.Sinr, uePCell.AvailPrbsDl)
	// TODO: define how to update when CA
	ue.FiveQi = signal.GetCQI(uePCell.Sinr)

	for index := range ue.NeighborCells {
		nCell := e.Model.Cells[strconv.FormatUint(uint64(ue.NeighborCells[index].NCGI), 10)]
		ue.NeighborCells[index].Rsrp = signal.RSRP(ue, nCell)
		ue.NeighborCells[index].Sinr = signal.Sinr(ue.Location, ue.Height, nCell, utils.GetNeighborCells(nCell, e.Model.Cells))
		ue.NeighborCells[index].Rsrq = signal.RSRQ(ue.NeighborCells[index].Sinr, ue.NeighborCells[index].AvailPrbsDl)
	}
	ueCopy := *ue
	e.Model.UEs[strconv.FormatUint(uint64(ue.IMSI), 10)] = &ueCopy
}
