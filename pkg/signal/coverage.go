package signal

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/davidkleiven/gononlin/nonlin"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
)

type FProvider func(x0 []float64) (f func(out, x []float64))

// Runs Newton Krylov solver to compute the signal coverage points
func ComputePointsWithNewtonKrylov(fp FProvider, guessChan <-chan []float64, maxIter int) <-chan model.Coordinate {

	pointsChannel := make(chan model.Coordinate)
	var wg sync.WaitGroup
	numWorkers := 15

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			solver := nonlin.NewtonKrylov{
				// Maximum number of Newton iterations
				Maxiter: maxIter,

				// Stepsize used to approximate jacobian with finite differences
				StepSize: 1e-4,

				// Tolerance for the solution
				Tol: 1e-6,

				// Stencil for Jacobian
				// Stencil: 8,
			}

			for x0 := range guessChan {
				problem := nonlin.Problem{
					F: fp(x0),
				}
				res, err := solver.Solve(problem, x0)
				if err != nil {
					continue
				}
				xInDomain := res.X[0] > 0 && res.X[1] > 0 && math.Abs(res.X[0]) <= 90 && math.Abs(res.X[1]) <= 180
				if res.Converged && xInDomain {
					pointsChannel <- model.Coordinate{Lat: res.X[0], Lng: res.X[1]}
				}
			}

		}()
	}

	go func() {
		defer close(pointsChannel)
		wg.Wait()
	}()

	return pointsChannel
}

func GetRandGuessesChan(cell model.Cell, numGuesses int) <-chan []float64 {
	rgChan := make(chan []float64)

	distance_to_deg := map[string]float64{"1m": 0.00001, "55m": 0.0005, "3.3km": 0.03}

	go func() {
		defer close(rgChan)
		for i := 1; i < numGuesses; i++ {

			offset := distance_to_deg["55m"] + math.Min(float64(i)*distance_to_deg["1m"]*rand.Float64(), distance_to_deg["3.3km"])

			latSign := (rand.Float64() - 0.5) * 2
			longSign := (rand.Float64() - 0.5) * 2

			guess := []float64{cell.Sector.Center.Lat + (latSign * offset), cell.Sector.Center.Lng + (longSign * offset)}
			select {
			case rgChan <- guess:
			default:
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()
	return rgChan
}

func GetGuessesChan(guessesCoord []model.Coordinate) <-chan []float64 {
	gChan := make(chan []float64, len(guessesCoord))
	go func() {
		defer close(gChan)
		for _, guessCoord := range guessesCoord {
			guess := []float64{guessCoord.Lat, guessCoord.Lng}
			select {
			case gChan <- guess:
			default:
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()
	return gChan
}

func RadiationPatternF(ueHeight float64, cell *model.Cell, refSignalStrength float64) (f func(out, x []float64)) {
	return func(out, x []float64) {
		coord := model.Coordinate{Lat: x[0], Lng: x[1]}
		fValue := RadiatedStrength(coord, ueHeight, *cell) - refSignalStrength
		out[0] = fValue
		out[1] = fValue
	}
}
func CoverageF(ueHeight float64, cell *model.Cell, refSignalStrength, mpf float64, radiationPatternBoundary []model.Coordinate) (f func(out, x []float64)) {
	return func(out, x []float64) {
		coord := model.Coordinate{Lat: x[0], Lng: x[1]}
		fValue := Strength(coord, ueHeight, mpf, *cell) - refSignalStrength
		out[0] = fValue
		out[1] = fValue
	}
}

func GetRPBoundaryPoints(ueHeight float64, cell *model.Cell, refSignalStrength float64) []model.Coordinate {
	rpFp := func(x0 []float64) (f func(out, x []float64)) {
		return RadiationPatternF(ueHeight, cell, refSignalStrength)
	}
	rpBoundaryPointsCh := ComputePointsWithNewtonKrylov(rpFp, GetRandGuessesChan(*cell, 3000), 60)
	rpBoundaryPoints := make([]model.Coordinate, 0)
	for rpBp := range rpBoundaryPointsCh {
		rpBoundaryPoints = append(rpBoundaryPoints, rpBp)
	}
	return utils.SortCoordinatesByBearing(cell.Sector.Center, rpBoundaryPoints)
}

func GetCovBoundaryPoints(ueHeight float64, cell *model.Cell, refSignalStrength float64, rpBoundaryPoints []model.Coordinate) []model.Coordinate {
	cfp := func(x0 []float64) (f func(out, x []float64)) {
		mpf := RiceanFading(GetRiceanK(cell))
		return CoverageF(ueHeight, cell, refSignalStrength, mpf, rpBoundaryPoints)
	}
	covBoundaryPointsCh := ComputePointsWithNewtonKrylov(cfp, GetGuessesChan(rpBoundaryPoints), 100)
	covBoundaryPoints := make([]model.Coordinate, 0)
	for cbp := range covBoundaryPointsCh {
		covBoundaryPoints = append(covBoundaryPoints, cbp)
	}
	return utils.SortCoordinatesByBearing(cell.Sector.Center, covBoundaryPoints)
}
