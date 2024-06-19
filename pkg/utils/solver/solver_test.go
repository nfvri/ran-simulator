package solver

import (
	"fmt"
	"math"
	"testing"

	"github.com/davidkleiven/gononlin/nonlin"
	"github.com/nfvri/ran-simulator/pkg/mobility"
	"github.com/nfvri/ran-simulator/pkg/model"
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
		Sector: model.Sector{
			Azimuth: 21,
			Center:  model.Coordinate{Lat: 37.979207, Lng: 23.716702},
			Height:  30,
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
	problem := nonlin.Problem{
		F: func(out, x []float64) {
			height := 1.5
			coord := model.Coordinate{Lat: x[0], Lng: x[1]}
			out[0] = 87 + mobility.StrengthAtLocation(coord, height, cell)
			out[1] = 87 + mobility.StrengthAtLocation(coord, height, cell)
		},
	}

	solver := nonlin.NewtonKrylov{
		// Maximum number of Newton iterations
		Maxiter: 1000,

		// Stepsize used to appriximate jacobian with finite differences
		StepSize: 1e-3,

		// Tolerance for the solution
		Tol: 1e-7,

		// Stencil for Jacobian
		Stencil: 8,
	}

	results := make([]nonlin.Result, 0)
	for outerPoint := -0.1; outerPoint <= 0.1; outerPoint += 0.03 {
		x0 := []float64{37.979207 - outerPoint, 23.716702 - outerPoint}
		res := solver.Solve(problem, x0)
		if res.Converged {
			results = append(results, res)
		}
		x1 := []float64{37.979207 + outerPoint, 23.716702 - outerPoint}
		res = solver.Solve(problem, x1)
		if res.Converged {
			results = append(results, res)
		}
		x2 := []float64{37.979207 - outerPoint, 23.716702 + outerPoint}
		res = solver.Solve(problem, x2)
		if res.Converged {
			results = append(results, res)
		}
		x3 := []float64{37.979207 + outerPoint, 23.716702 + outerPoint}
		res = solver.Solve(problem, x3)
		if res.Converged {
			results = append(results, res)
		}
	}
	t.Logf("results length: %d", len(results))
	for _, result := range results {
		t.Logf("Roots: (x, y) = %v Function values: %v \n", result.X, result.F)
	}

	// Output:
	//
	// Root: (x, y) = (1.00, 2.00)
	// Function value: (-0.00, 0.00)
}
