package signal

import (
	"math/rand"

	"github.com/davidkleiven/gononlin/nonlin"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
	log "github.com/sirupsen/logrus"
)

// Runs Newton Krylov solver to compute the signal coverage points
func GetSignalCoverageNewtonKrylov(cell model.Cell, ueHeight float64) []model.Coordinate {

	// cell = model.Cell{
	// 	TxPowerDB: 40,
	// 	CellType:  types.CellType_MACRO,
	// 	Sector: model.Sector{
	// 		Azimuth: 0,
	// 		Center:  model.Coordinate{Lat: 37.979207, Lng: 23.716702},
	// 		Height:  30,
	// 		Arc:     90,
	// 		Tilt:    0,
	// 	},
	// 	Channel: model.Channel{
	// 		Environment:  "urban",
	// 		LOS:          true,
	// 		SSBFrequency: 900,
	// 	},
	// 	Beam: model.Beam{
	// 		H3dBAngle:              65,
	// 		V3dBAngle:              65,
	// 		MaxGain:                8,
	// 		MaxAttenuationDB:       30,
	// 		VSideLobeAttenuationDB: 30,
	// 	},
	// }
	problem := nonlin.Problem{
		F: func(out, x []float64) {
			coord := model.Coordinate{Lat: x[0], Lng: x[1]}
			out[0] = 87 + StrengthAtLocation(coord, ueHeight, cell)
			out[1] = 87 + StrengthAtLocation(coord, ueHeight, cell)
		},
	}

	solver := nonlin.NewtonKrylov{
		// Maximum number of Newton iterations
		Maxiter: 10,

		// Stepsize used to appriximate jacobian with finite differences
		StepSize: 1e-4,

		// Tolerance for the solution
		Tol: 1e-7,

		// Stencil for Jacobian
		// Stencil: 8,
	}

	guesses := make([][]float64, 0)
	results := make([]nonlin.Result, 0)
	for i := 360; i < 900; i++ {
		outerPoint := float64(i) * 0.0005 * rand.Float64()
		sign1 := rand.Float64() - 0.5
		sign2 := rand.Float64() - 0.5
		x0 := []float64{cell.Sector.Center.Lat + (sign1 * outerPoint), cell.Sector.Center.Lng + (sign2 * outerPoint)}
		res := solver.Solve(problem, x0)
		if res.Converged {
			guesses = append(guesses, x0)
			results = append(results, res)
		}
	}
	log.Debugf("results length: %d", len(results))
	for i, result := range results {
		log.Debugf("Roots: (x, y) = %v Function values: %v Guesses: %v \n", result.X, result.F, guesses[i])
	}

	coords := make([]model.Coordinate, 0)
	for _, result := range results {
		hCoords := model.Coordinate{
			Lat: result.X[0],
			Lng: result.X[1],
		}
		coords = append(coords, hCoords)
	}

	sortedCoords := utils.SortCoordinatesByBearing(cell.Sector.Center, coords)

	for _, sortedCoord := range sortedCoords {
		log.Debugf("[%f, %f], \n", sortedCoord.Lat, sortedCoord.Lng)
	}

	return sortedCoords
}
