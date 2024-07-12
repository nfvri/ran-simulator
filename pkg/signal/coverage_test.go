package signal

import (
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
)

func TestStrengthAtLocationNewtonKrylov(t *testing.T) {

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
	coverageF := func(out, x []float64) {
		coord := model.Coordinate{Lat: x[0], Lng: x[1]}
		out[0] = StrengthAtLocation(coord, ueHeight, cell) - refSignalStrength
		out[1] = StrengthAtLocation(coord, ueHeight, cell) - refSignalStrength
	}
	sortedCoords := ComputeCoverageNewtonKrylov(cell, coverageF)

	for _, sortedCoord := range sortedCoords {
		t.Logf("[%f, %f], \n", sortedCoord.Lat, sortedCoord.Lng)
	}
	// Output:
	//
	// Root: (x, y) = (1.00, 2.00)
	// Function value: (-0.00, 0.00)
}
