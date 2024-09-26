// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package handover

import (
	"context"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/store/cells"
	"github.com/nfvri/ran-simulator/pkg/store/ues"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var logHoCtrl = logging.GetLogger("handover", "controller")

// NewHOController returns the hanover controller
func NewHOController(hoType HOType, cellStore cells.Store, ueStore ues.Store) HOController {
	return &hoController{
		hoType:     hoType,
		cellStore:  cellStore,
		ueStore:    ueStore,
		inputChan:  make(chan model.UE),
		outputChan: make(chan A3HandoverDecision),
	}
}

// HOController is an abstraction of the handover controller
type HOController interface {
	// Start starts handover controller
	Start(ctx context.Context)

	// GetInputChan returns input channel
	GetInputChan() chan model.UE

	// GetOutputChan returns output channel
	GetOutputChan() chan A3HandoverDecision
}

// HOType is the type of hanover - currently it is string
// ToDo: define enumerated handover type into rrm-son-lib
type HOType string

type hoController struct {
	cellStore  cells.Store
	ueStore    ues.Store
	hoType     HOType
	inputChan  chan model.UE
	outputChan chan A3HandoverDecision
}

func (h *hoController) Start(ctx context.Context) {
	switch h.hoType {
	case "A3":
		// h.startA3HandoverHandler(ctx)
		h.startA3HandoverHandler(ctx)
	}
}

func (h *hoController) startA3HandoverHandler(ctx context.Context) {
	logHoCtrl.Info("Handover controller starting with A3HandoveHandler")
	handler := NewA3Handover()

	go handler.Start()
	// for input
	go h.forwardReportToA3HandoverHandler(handler)
	//for output
	go h.forwardHandoverDecision(handler)
}

func (h *hoController) forwardReportToA3HandoverHandler(handler A3Handover) {
	for ue := range h.inputChan {
		logHoCtrl.Debugf("[input] Measurement report for HO decision: %v", ue)
		handler.PushMeasurementEventA3(ue)
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

func (h *hoController) GetOutputChan() chan A3HandoverDecision {
	return h.outputChan
}
