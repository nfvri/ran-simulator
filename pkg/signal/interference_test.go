package signal

import (
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestCalculateRSRQ(t *testing.T) {
	numPRBs := 24
	rsrpDbm := -90.0
	sinrDbm := -3.0

	rssiDbm := RSSI(rsrpDbm, sinrDbm)

	sinrCalc := SINR(rsrpDbm, rssiDbm)

	rsrq := RSRQ(sinrDbm, numPRBs)
	rsrq1 := RSRQ1(rsrpDbm, sinrDbm, numPRBs)
	rsrqCalc := RSRQ(sinrCalc, numPRBs)

	assert.Equal(t, rsrq, rsrq1)
	assert.Equal(t, rsrq, rsrqCalc)

	t.Logf("rssiDbm: %f sinrCalc:%v\n", rsrq, sinrCalc)
	t.Logf("rsrq: %f rsrq1:%v  rsrq1Calc: %v \n", rsrq, rsrq1, rsrqCalc)

}

func TestUpdateCells(t *testing.T) {
	cqi := 5
	numPRBs := 24
	sinr := GetSINR(cqi)
	rsrp := -95.0
	rsrq := RSRQ(sinr, numPRBs)
	CreateSimulationUE(17660905553922, 1, cqi, sinr, rsrp, rsrq, model.Coordinate{Lat: 0.0, Lng: 0.0}, []*model.UECell{})
}
