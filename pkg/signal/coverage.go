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

// Runs Newton Krylov solver to solve the function provided by fp
func ComputePoints(fp FProvider, guessChan <-chan []float64, nonLinSolver nonlin.NewtonKrylov, stop *bool) <-chan model.Coordinate {

	pointsChannel := make(chan model.Coordinate)
	var wg sync.WaitGroup
	numWorkers := 30

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			solver := nonLinSolver

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
	cutOffDistance := utils.MetersToLatDegrees(5000)

	latScalingFactor := utils.DegreesToMeters(cell.BoundingBox.MaxLat-cell.BoundingBox.MinLat) * 0.01
	lngScalingFactor := utils.DegreesToMeters(cell.BoundingBox.MaxLng-cell.BoundingBox.MinLng) * 0.01

	centerLat := (cell.BoundingBox.MinLat + cell.BoundingBox.MaxLat) / 2.0
	centerLng := (cell.BoundingBox.MinLng + cell.BoundingBox.MaxLng) / 2.0
	go func() {
		defer close(rgChan)
		for j := 0; j < 5; j++ {
			for i := 0.0; i < float64(numGuesses)/5.0; i++ {

				offsetLat := math.Min(i*step*rand.Float64(), cutOffDistance)
				offsetLng := math.Min(i*step*rand.Float64(), cutOffDistance)

				repositionLat := (rand.Float64() - 0.5) * 2 * latScalingFactor / float64(cqi)
				repositionLng := (rand.Float64() - 0.5) * 2 * lngScalingFactor / float64(cqi)

				guess := []float64{centerLat + (repositionLat * offsetLat), centerLng + (repositionLng * offsetLng)}
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

func GetRandGuessesChanCells(cell model.Cell, numGuesses, stepSizeMeters, initOffsetMeters, cutOffDistanceMeters float64) <-chan []float64 {
	rgChan := make(chan []float64)

	stepSize := utils.MetersToLatDegrees(float64(stepSizeMeters))
	initOffset := utils.MetersToLatDegrees(initOffsetMeters)
	cutOffDistance := utils.MetersToLatDegrees(cutOffDistanceMeters)

	go func() {
		defer close(rgChan)
		for j := 0; j < 3; j++ {
			for i := 0.0; i < numGuesses/3.0; i++ {

				offsetLat := math.Min(i*stepSize*rand.Float64(), cutOffDistance)
				offsetLong := math.Min(i*stepSize*rand.Float64(), cutOffDistance)

				latSign := (rand.Float64() - 0.5) * 2
				longSign := (rand.Float64() - 0.5) * 2

				guess := []float64{cell.Sector.Center.Lat + (latSign * (initOffset + offsetLat)), cell.Sector.Center.Lng + (longSign * (initOffset + offsetLong))}
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

	// TODO: add cell.Channel.SSBFrequency in equation
	// maxIter * stepSizeKrylof >= cutOffDistance + initOffset
	numGuesses := 5000.0
	maxIter := 300
	cutOffDistance := -136*refSignalStrength - 11000
	stepSizeMeters := (2 * cutOffDistance) / float64(maxIter)
	initOffset := cutOffDistance / 5
	stepsize := (20 * cutOffDistance) / numGuesses
	log.Infof("cutOffDistance: %v -- initOffset: %v -- stepsize: %v -- stepSizeKrylof: %v", cutOffDistance, initOffset, stepsize, stepSizeMeters)
	guessChan := GetRandGuessesChanCells(*cell, numGuesses, stepsize, initOffset, cutOffDistance)
	newtonKrylovSolver := nonlin.NewtonKrylov{
		// Maximum number of Newton iterations
		Maxiter: maxIter,

		// Stepsize used to approximate jacobian with finite differences
		StepSize: utils.MetersToLatDegrees(stepSizeMeters),

		// Tolerance for the solution
		Tol: 0.01,

		// Stencil for Jacobian
		// Stencil: 8,
	}
	stop := false
	rpBoundaryPointsCh := ComputePoints(rpFp, guessChan, newtonKrylovSolver, &stop)
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
	maxIter := 200
	stepSizeMeters := 10.0
	newtonKrylovSolver := nonlin.NewtonKrylov{
		// Maximum number of Newton iterations
		Maxiter: maxIter,

		// Stepsize used to approximate jacobian with finite differences
		StepSize: utils.MetersToLatDegrees(stepSizeMeters),

		// Tolerance for the solution
		Tol: 0.1,

		// Stencil for Jacobian
		// Stencil: 8,
	}
	stop := false
	covBoundaryPointsCh := ComputePoints(cfp, GetGuessesChan(rpBoundaryPoints), newtonKrylovSolver, &stop)
	covBoundaryPoints := make([]model.Coordinate, 0)
	for cbp := range covBoundaryPointsCh {
		covBoundaryPoints = append(covBoundaryPoints, cbp)
	}
	return utils.SortCoordinatesByBearing(cell.Sector.Center, covBoundaryPoints)
}
