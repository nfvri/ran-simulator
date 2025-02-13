package ues

import (
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/signal"
)

func TestUpdateCells(t *testing.T) {
	ueHeight := 1.5
	cqi := 5
	numPRBs := 24
	sinr := signal.GetSINR(cqi)
	rsrp := -95.0
	rsrq := signal.RSRQ(sinr, numPRBs)
	beamQS := model.BeamQS{
		BeamID: model.BeamID{NCGI: 1234, CarrierIndex: 1, BeamIndex: 1},
		CQI:    1,
	}
	CreateSimulationUE(17660905553922, beamQS, 1, numPRBs, ueHeight, sinr, rsrp, rsrq, model.Coordinate{Lat: 0.0, Lng: 0.0}, []*model.UECell{})
}
