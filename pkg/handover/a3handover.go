// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package handover

import (
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var logA3ho = logging.GetLogger()

// A3Handover is an abstraction of A3 handover
type A3Handover interface {
	// Start starts the A3 handover module
	Start()

	// GetInputChan returns the channel to push measurement
	GetInputChan() chan *model.UE

	// GetOutputChan returns the channel to get handover event
	GetOutputChan() chan HandoverDecision

	// PushMeasurementEventA3 pushes measurement to the input channel
	PushMeasurementEventA3(*model.UE)
}

type a3Handover struct {
	a3HandoverHandler *A3HandoverHandler
}

// NewA3Handover returns an A3 handover object
func NewA3Handover(handler *A3HandoverHandler) A3Handover {
	return &a3Handover{
		// a3HandoverHandler: handover.NewA3HandoverHandler(),
		a3HandoverHandler: handler,
	}
}

func (h *a3Handover) Start() {
	logA3ho.Info("A3 handover handler starting")
	go h.a3HandoverHandler.Run()
}

func (h *a3Handover) GetInputChan() chan *model.UE {
	return h.a3HandoverHandler.Chans.InputChan
}

func (h *a3Handover) GetOutputChan() chan HandoverDecision {
	return h.a3HandoverHandler.Chans.OutputChan
}

func (h *a3Handover) PushMeasurementEventA3(ue *model.UE) {
	h.a3HandoverHandler.Chans.InputChan <- ue
}
