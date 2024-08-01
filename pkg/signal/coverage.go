package signal

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/davidkleiven/gononlin/nonlin"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
	log "github.com/sirupsen/logrus"
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

// Runs Newton Krylov solver to compute the signal coverage points
func ComputePointsWithNewtonKrylovUEs(fp FProvider, guessChan <-chan []float64, maxIter int, stop *bool) <-chan model.Coordinate {

	pointsChannel := make(chan model.Coordinate)
	var wg sync.WaitGroup
	numWorkers := 30

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			solver := nonlin.NewtonKrylov{
				Maxiter:  maxIter,
				StepSize: 5e-5,
				Tol:      0.5,
			}

			for x0 := range guessChan {
				if *stop {
					break
				}
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

func GetRandGuessesChanUEs(cell model.Cell, numGuesses, cqi, stepMeters int) <-chan []float64 {
	rgChan := make(chan []float64)

	step := utils.MetersToLatDegrees(float64(stepMeters))
	initOffset := utils.MetersToLatDegrees(50)
	cutOffDistance := utils.MetersToLatDegrees(5000)

	go func() {
		defer close(rgChan)
		for j := 0; j < 5; j++ {
			for i := 0; i < numGuesses/5; i++ {

				offsetLat := initOffset + math.Min((float64(i))*step*rand.Float64(), cutOffDistance)
				offsetLong := initOffset + math.Min((float64(i))*step*rand.Float64(), cutOffDistance)

				repositionLat := (rand.Float64() - 0.5) * 2 / float64(cqi)

				repositionLong := (rand.Float64() - 0.5) * 2 / float64(cqi)

				guess := []float64{cell.Sector.Center.Lat + (repositionLat * offsetLat), cell.Sector.Center.Lng + (repositionLong * offsetLong)}
				select {
				case rgChan <- guess:
				default:
					time.Sleep(1 * time.Millisecond)
				}
			}
		}
	}()
	return rgChan
}

func GetRandGuessesChanCells(cell model.Cell, numGuesses, stepMeters int) <-chan []float64 {
	rgChan := make(chan []float64)

	step := utils.MetersToLatDegrees(float64(stepMeters))
	initOffset := utils.MetersToLatDegrees(50)
	cutOffDistance := utils.MetersToLatDegrees(5000)

	go func() {
		defer close(rgChan)
		for i := 0; i < numGuesses; i++ {

			offsetLat := initOffset + math.Min(float64(i)*step*rand.Float64(), cutOffDistance)
			offsetLong := initOffset + math.Min(float64(i)*step*rand.Float64(), cutOffDistance)

			latSign := (rand.Float64() - 0.5) * 2
			longSign := (rand.Float64() - 0.5) * 2

			guess := []float64{cell.Sector.Center.Lat + (latSign * offsetLat), cell.Sector.Center.Lng + (longSign * offsetLong)}
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
	log.Debugf("calculating radiation pattern for cell:%v", cell.NCGI)
	rpFp := func(x0 []float64) (f func(out, x []float64)) {
		return RadiationPatternF(ueHeight, cell, refSignalStrength)
	}
	rpBoundaryPointsCh := ComputePointsWithNewtonKrylov(rpFp, GetRandGuessesChanCells(*cell, 3000, 50), 60)
	rpBoundaryPoints := make([]model.Coordinate, 0)
	for rpBp := range rpBoundaryPointsCh {
		rpBoundaryPoints = append(rpBoundaryPoints, rpBp)
	}
	return utils.SortCoordinatesByBearing(cell.Sector.Center, rpBoundaryPoints)
}

func GetCovBoundaryPoints(ueHeight float64, cell *model.Cell, refSignalStrength float64, rpBoundaryPoints []model.Coordinate) []model.Coordinate {
	log.Debugf("calculating coverage for cell:%v", cell.NCGI)
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
