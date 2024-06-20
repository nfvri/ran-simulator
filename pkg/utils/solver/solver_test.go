package solver

import (
	"fmt"
	"math"
	"testing"

	"github.com/davidkleiven/gononlin/nonlin"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
)

func TestExampleNewtonKrylov(t *testing.T) {
	// This example shows how one can use NewtonKrylov to solve the
	// system of equations
	// (x-1)^2*(x - y) = 0
	// (x-2)^3*cos(2*x/y) = 0

	problem := nonlin.Problem{
		F: func(out, x []float64) {
			out[0] = math.Pow(x[0]-1.0, 2.0) * (x[0] - x[1])
			out[1] = math.Pow(x[1]-2.0, 3.0) * math.Cos(2.0*x[0]/x[1])
		},
	}

	solver := nonlin.NewtonKrylov{
		// Maximum number of Newton iterations
		Maxiter: 1000,

		// Stepsize used to appriximate jacobian with finite differences
		StepSize: 1e-4,

		// Tolerance for the solution
		Tol: 1e-7,
	}

	x0 := []float64{0.0, 3.0}
	res := solver.Solve(problem, x0)
	fmt.Printf("Root: (x, y) = (%.2f, %.2f)\n", res.X[0], res.X[1])
	fmt.Printf("Function value: (%.2f, %.2f)\n", res.F[0], res.F[1])

	// Output:
	//
	// Root: (x, y) = (1.00, 2.00)
	// Function value: (-0.00, 0.00)
}

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
	sortedCoords := GetSignalCoverageNewtonKrylov(cell, ueHeight)

	for _, sortedCoord := range sortedCoords {
		t.Logf("[%f, %f], \n", sortedCoord.Lat, sortedCoord.Lng)
	}
	// Output:
	//
	// Root: (x, y) = (1.00, 2.00)
	// Function value: (-0.00, 0.00)
}
