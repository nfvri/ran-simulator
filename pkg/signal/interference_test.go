package signal

import (
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
)

func TestSinrAtLocationNewtonKrylov(t *testing.T) {
	// cell grid Points shoulf be initialized
	cell1 := model.Cell{
		NCGI:      17660905537537,
		TxPowerDB: 40,
		CellType:  types.CellType_MACRO,
		Sector: model.Sector{
			Center:  model.Coordinate{Lat: 37.981629, Lng: 23.743353},
			Azimuth: 90,
			Arc:     90,
			Tilt:    20,
			Height:  25,
		},
		Neighbors: []types.NCGI{17660905570307},
		Channel: model.Channel{
			Environment:  "urban",
			LOS:          true,
			SSBFrequency: 3200,
		},
		Beam: model.Beam{
			H3dBAngle:              90,
			V3dBAngle:              65,
			MaxGain:                8,
			MaxAttenuationDB:       40,
			VSideLobeAttenuationDB: 40,
		},
	}

	cell2 := model.Cell{
		NCGI:      17660905570307,
		TxPowerDB: 30,
		CellType:  types.CellType_MACRO,
		Sector: model.Sector{
			Center:  model.Coordinate{Lat: 37.969502, Lng: 23.796955},
			Azimuth: 0,
			Arc:     120,
			Tilt:    20,
			Height:  25,
		},
		Neighbors: []types.NCGI{17660905537537},
		Channel: model.Channel{
			Environment:  "urban",
			LOS:          true,
			SSBFrequency: 3200,
		},
		Beam: model.Beam{
			H3dBAngle:              120,
			V3dBAngle:              65,
			MaxGain:                8,
			MaxAttenuationDB:       40,
			VSideLobeAttenuationDB: 40,
		},
	}
	ueHeight := 1.5
	sinr := Sinr(model.Coordinate{Lat: 37.981659, Lng: 23.743323}, ueHeight, &cell1, []*model.Cell{&cell2})
	t.Logf("[%f], \n", sinr)

	// Output:
	//
	// Root: (x, y) = (1.00, 2.00)
	// Function value: (-0.00, 0.00)
}

func TestCalculateRSRQ(t *testing.T) {
	numPRBs := 24
	rsrpDbm := -10.0
	sinrDbm := -5.0

	rssiDbm := RSSI(rsrpDbm, sinrDbm)

	sinrCalc := SINR(rsrpDbm, rssiDbm)

	rsrq := RSRQ(rsrpDbm, sinrDbm, numPRBs)
	rsrq1 := RSRQ1(sinrDbm, numPRBs)
	rsrq1Calc := RSRQ1(sinrCalc, numPRBs)

	t.Logf("rssiDbm: %f sinrCalc:%v\n", rsrq, sinrCalc)
	t.Logf("rsrq: %f rsrq1:%v  rsrq1Calc: %v \n", rsrq, rsrq1, rsrq1Calc)

}
