// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package handover

import (
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-lib-go/pkg/logging"
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
	UE          *model.UE
	ServingCell *model.Cell
	TargetCell  *model.UECell
	Feasible    bool
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
	logHoCtrl.Info("Handover controller starting with A3HandoveHandler")
	go h.HoHandler.Start()
	// for input
	go h.forwardReportToA3HandoverHandler(h.HoHandler)
	//for output
	go h.forwardHandoverDecision(h.HoHandler)
}

func (h *hoController) forwardReportToA3HandoverHandler(handler A3Handover) {
	for ue := range h.inputChan {
		logHoCtrl.Debugf("[input] Measurement report for HO decision: %v", ue)
		handler.PushMeasurementEventA3(&ue)
	}
}

func (h *hoController) forwardHandoverDecision(handler A3Handover) {
	for hoDecision := range handler.GetOutputChan() {
		logHoCtrl.Debugf("[output] Handover decision: %v", hoDecision)
		h.outputChan <- hoDecision
	}
}

func (h *hoController) GetInputChan() chan model.UE {
	return h.inputChan
}

func (h *hoController) GetOutputChan() chan HandoverDecision {
	return h.outputChan
}
