package signal

import (
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
)

func TestStrengthAtLocationNewtonKrylov(t *testing.T) {
	// run with: go test -v -timeout 300s  ./pkg/utils/solver
	// This example shows how one can use NewtonKrylov to solve the
	// system of equations
	// (x-1)^2*(x - y) = 0
	// (x-2)^3*cos(2*x/y) = 0
	cell := model.Cell{
		TxPowerDB: 40,
		CellType:  types.CellType_MACRO,
		Sector: model.Sector{
			Azimuth: 0,
			Center:  model.Coordinate{Lat: 37.979207, Lng: 23.716702},
			Height:  30,
			Arc:     90,
			Tilt:    0,
		},
		Channel: model.Channel{
			Environment:  "urban",
			LOS:          true,
			SSBFrequency: 900,
		},
		Beam: model.Beam{
			H3dBAngle:              65,
			V3dBAngle:              65,
			MaxGain:                8,
			MaxAttenuationDB:       30,
			VSideLobeAttenuationDB: 30,
		},
	}

	ueHeight := 1.5
	const refSignalStrength = -87
	sortedCoords := ComputeCoverageNewtonKrylov(cell, ueHeight, refSignalStrength)

	for _, sortedCoord := range sortedCoords {
		t.Logf("[%f, %f], \n", sortedCoord.Lat, sortedCoord.Lng)
	}
	// Output:
	//
	// Root: (x, y) = (1.00, 2.00)
	// Function value: (-0.00, 0.00)
}
