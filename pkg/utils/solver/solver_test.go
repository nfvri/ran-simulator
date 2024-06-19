package solver

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"

	"github.com/davidkleiven/gononlin/nonlin"
	"github.com/nfvri/ran-simulator/pkg/mobility"
	"github.com/nfvri/ran-simulator/pkg/model"

	haversine "github.com/LucaTheHacker/go-haversine"
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
			Azimuth: 270,
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
			// height2 := 3.0
			coord := model.Coordinate{Lat: x[0], Lng: x[1]}
			out[0] = 87 + mobility.StrengthAtLocation(coord, height, cell)
			out[1] = 87 + mobility.StrengthAtLocation(coord, height, cell)
		},
	}

	solver := nonlin.NewtonKrylov{
		// Maximum number of Newton iterations
		Maxiter: 50,

		// Stepsize used to appriximate jacobian with finite differences
		StepSize: 1e-4,

		// Tolerance for the solution
		Tol: 1e-7,

		// Stencil for Jacobian
		// Stencil: 8,
	}

	guesses := make([][]float64, 0)
	results := make([]nonlin.Result, 0)
	for i := 100; i < 200; i++ {
		outerPoint := float64(i) * 0.00005
		sign1 := rand.Float64() - 0.5
		sign2 := rand.Float64() - 0.5
		x0 := []float64{cell.Sector.Center.Lat + (sign1 * outerPoint), cell.Sector.Center.Lng + (sign2 * outerPoint)}
		res := solver.Solve(problem, x0)
		if res.Converged {
			guesses = append(guesses, x0)
			results = append(results, res)
		}
	}
	t.Logf("results length: %d", len(results))
	for i, result := range results {
		t.Logf("Roots: (x, y) = %v Function values: %v Guesses: %v \n", result.X, result.F, guesses[i])
	}

	haversinneCoords := make([]haversine.Coordinates, 0)
	for _, result := range results {
		t.Logf("[%f, %f], \n", result.X[0], result.X[1])
		hCoords := haversine.Coordinates{
			Latitude:  result.X[0],
			Longitude: result.X[1],
		}
		haversinneCoords = append(haversinneCoords, hCoords)
	}

	// Sorting a slice of coords by haversine distance
	sort.Slice(haversinneCoords, func(i, j int) bool {
		center := haversine.Coordinates{
			Latitude:  cell.Sector.Center.Lat,
			Longitude: cell.Sector.Center.Lng,
		}
		return haversine.Distance(center, haversinneCoords[i]).Kilometers() < haversine.Distance(center, haversinneCoords[j]).Kilometers()
	})

	for _, sortedHaversinneCoord := range haversinneCoords {
		t.Logf("[%f, %f], \n", sortedHaversinneCoord.Latitude, sortedHaversinneCoord.Longitude)
	}
	// Output:
	//
	// Root: (x, y) = (1.00, 2.00)
	// Function value: (-0.00, 0.00)
}
