package signal

import (
	"fmt"
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
	rpFp := func(x0 []float64) (f func(out, x []float64)) {
		return RadiationPatternF(ueHeight, &cell, refSignalStrength)
	}
	rpBoundaryPointsCh := ComputeCoverageNewtonKrylov(rpFp, GetRandGuessesChan(cell), 10)

	for rpBoundaryPoint := range rpBoundaryPointsCh {
		t.Logf("[%f, %f], \n", rpBoundaryPoint.Lat, rpBoundaryPoint.Lng)
	}
	// Output:
	//
	// Root: (x, y) = (1.00, 2.00)
	// Function value: (-0.00, 0.00)
}

func TestStrength(t *testing.T) {
	cell := model.Cell{
		TxPowerDB: 40,
		CellType:  types.CellType_MACRO,
		Sector: model.Sector{
			Azimuth: 90,
			Center:  model.Coordinate{Lat: 37.981629, Lng: 23.743353},
			Height:  0,
			Arc:     90,
			Tilt:    20,
		},
		Channel: model.Channel{
			Environment:  "urban",
			LOS:          false,
			SSBFrequency: 900,
		},
		Beam: model.Beam{
			H3dBAngle:              90,
			V3dBAngle:              65,
			MaxGain:                8,
			MaxAttenuationDB:       40,
			VSideLobeAttenuationDB: 40,
		},
	}

	coord := model.Coordinate{Lat: 87.63223356680056, Lng: 73.40325326694467}
	mpf := 0.3638433520844825
	s := Strength(coord, 1.5, mpf, cell)
	fmt.Printf("s: %v", s)
}
