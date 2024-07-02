package signal

import (
	"math"
	"math/rand"

	"github.com/davidkleiven/gononlin/nonlin"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
	log "github.com/sirupsen/logrus"
)

// Runs Newton Krylov solver to compute the signal coverage points
func ComputeCoverageNewtonKrylov(cell model.Cell, ueHeight float64, refSignalStrength float64) []model.Coordinate {

	problem := nonlin.Problem{
		F: func(out, x []float64) {
			coord := model.Coordinate{Lat: x[0], Lng: x[1]}
			out[0] = StrengthAtLocation(coord, ueHeight, cell) - refSignalStrength
			out[1] = StrengthAtLocation(coord, ueHeight, cell) - refSignalStrength
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
	boundaryPoints := make([]model.Coordinate, 0)
	for i := 360; i < 900; i++ {
		outerPoint := float64(i) * 0.0005 * rand.Float64()
		sign1 := rand.Float64() - 0.5
		sign2 := rand.Float64() - 0.5
		x0 := []float64{cell.Sector.Center.Lat + (sign1 * outerPoint), cell.Sector.Center.Lng + (sign2 * outerPoint)}
		// log.Infof("\n======================================\n")
		// log.Infof("\tcenter: (%v,%v)\n\t\tx0: (%v)\n", cell.Sector.Center.Lat, cell.Sector.Center.Lng, x0)

		log.Debugf("\n======================================\n")
		log.Debugf("\n\t\tx0: %v", x0)
		res := solver.Solve(problem, x0)
		log.Debugf("\t\n res: %v\n\t\tx0: %v", res, x0)
		if res.Converged {
			guesses = append(guesses, x0)
			results = append(results, res)
			if math.Abs(res.X[0]) > 90 || math.Abs(res.X[1]) > 180 {
				log.Debugf("\tcenter: (%v,%v)\n\t\toutlier: (%v,%v)", cell.Sector.Center.Lat, cell.Sector.Center.Lng, res.X[0], res.X[1])
			} else {
				boundaryPoints = append(boundaryPoints, model.Coordinate{
					Lat: res.X[0],
					Lng: res.X[1],
				})
			}

		}
	}
	if len(boundaryPoints) == 0 {
		log.Errorf("did not Converge")
	}
	log.Debugf("results length: %d", len(results))
	for i, result := range results {
		log.Debugf("Roots: (x, y) = %v Function values: %v Guesses: %v \n", result.X, result.F, guesses[i])
	}

	return utils.SortCoordinatesByBearing(cell.Sector.Center, boundaryPoints)
}
