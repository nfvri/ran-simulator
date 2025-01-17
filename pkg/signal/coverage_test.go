package signal

import (
	"fmt"
	"testing"

	"github.com/davidkleiven/gononlin/nonlin"
	"github.com/nfvri/onos-api/go/onos/ransim/types"
	"github.com/nfvri/ran-simulator/pkg/model"
)

func TestStrengthAtLocationNewtonKrylov(t *testing.T) {

	cell := &model.Cell{
		CellType: types.CellType_MACRO,
		CellConfig: model.CellConfig{
			Carriers: []*model.Carrier{
				{
					Beams: []*model.Beam{
						{
							H3dBAngle: 65,
							V3dBAngle: 65,
							MaxGain:   8,
							Azimuth:   0,
							Tilt:      0,
						},
					},
					TxPowerDB:              40,
					Center:                 model.Coordinate{Lat: 37.979207, Lng: 23.716702},
					Height:                 30,
					VSideLobeAttenuationDB: 30,
					ArfcnDL:                180000,
					Environment:            "urban",
					LOS:                    true,
				},
			},
		},
	}

	ueHeight := 1.5
	refSignalStrength := -87.0

	carrier := cell.Carriers[0]
	rpFp := func(x0 []float64) (f func(out, x []float64)) {
		return RadiationPatternF(cell, carrier, 0, ueHeight, refSignalStrength)
	}
	newtonKrylovSolver := nonlin.NewtonKrylov{
		// Maximum number of Newton iterations
		Maxiter: 100,

		// Stepsize used to approximate jacobian with finite differences
		StepSize: 10,

		// Tolerance for the solution
		Tol: 1e-6,

		// Stencil for Jacobian
		// Stencil: 8,
	}
	stop := false
	rpBoundaryPointsCh := ComputePoints(rpFp, GetRandGuessesChanCells(carrier, 3000, 10, 200, 1000), newtonKrylovSolver, &stop)

	for rpBoundaryPoint := range rpBoundaryPointsCh {
		t.Logf("[%f, %f], \n", rpBoundaryPoint.Lat, rpBoundaryPoint.Lng)
	}
	// Output:
	//
	// Root: (x, y) = (1.00, 2.00)
	// Function value: (-0.00, 0.00)
}

func TestStrength(t *testing.T) {
	cell := &model.Cell{
		CellType: types.CellType_MACRO,
		CellConfig: model.CellConfig{
			Carriers: []*model.Carrier{
				{
					Beams: []*model.Beam{
						{
							H3dBAngle: 90,
							V3dBAngle: 65,
							MaxGain:   8,
							Azimuth:   90,
							Tilt:      20,
						},
					},
					TxPowerDB:              40,
					Center:                 model.Coordinate{Lat: 37.981629, Lng: 23.743353},
					Height:                 30,
					VSideLobeAttenuationDB: 30,
					ArfcnDL:                180000,
					Environment:            "urban",
					LOS:                    false,
				},
			},
		},
	}

	coord := model.Coordinate{Lat: 87.63223356680056, Lng: 73.40325326694467}
	mpf := 0.3638433520844825
	beamID := model.BeamID{NCGI: cell.NCGI, CarrierIndex: 0, BeamIndex: 0}
	s := Strength(coord, 1.5, mpf, cell, beamID)
	fmt.Printf("s: %v", s)
}
