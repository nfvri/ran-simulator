package ues

import (
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/signal"
)

func TestUpdateCells(t *testing.T) {
	cqi := 5
	numPRBs := 24
	sinr := signal.GetSINR(cqi)
	rsrp := -95.0
	rsrq := signal.RSRQ(sinr, numPRBs)
	CreateSimulationUE(17660905553922, 1, cqi, numPRBs, sinr, rsrp, rsrq, model.Coordinate{Lat: 0.0, Lng: 0.0}, []*model.UECell{})
}
