package signal

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/davidkleiven/gononlin/nonlin"
	"github.com/nfvri/ran-simulator/pkg/model"
)

// Runs Newton Krylov solver to compute the signal coverage points
func ComputeCoverageNewtonKrylov(cell model.Cell, f func(out, x []float64), guessChan <-chan []float64, maxIter int) <-chan model.Coordinate {

	problem := nonlin.Problem{
		F: f,
	}

	boundaryPointsCh := make(chan model.Coordinate)
	var wg sync.WaitGroup
	numWorkers := 10

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			solver := nonlin.NewtonKrylov{
				// Maximum number of Newton iterations
				Maxiter: maxIter,

				// Stepsize used to approximate jacobian with finite differences
				StepSize: 1e-3,

				// Tolerance for the solution
				Tol: 1e-5,

				// Stencil for Jacobian
				// Stencil: 8,
			}

			for x0 := range guessChan {
				res := solver.Solve(problem, x0)
				xInDomain := math.Abs(res.X[0]) <= 90 && math.Abs(res.X[1]) <= 180
				if res.Converged && xInDomain {
					boundaryPointsCh <- model.Coordinate{Lat: res.X[0], Lng: res.X[1]}
				}
			}

		}()
	}

	go func() {
		defer close(boundaryPointsCh)
		wg.Wait()
	}()

	return boundaryPointsCh
}

func GetRandGuessesChan(cell model.Cell) <-chan []float64 {
	rgChan := make(chan []float64)
	go func() {
		defer close(rgChan)
		for i := 360; i < 900; i++ {
			outerPoint := float64(i) * 0.0005 * rand.Float64()
			sign1 := rand.Float64() - 0.5
			sign2 := rand.Float64() - 0.5
			guess := []float64{cell.Sector.Center.Lat + (sign1 * outerPoint), cell.Sector.Center.Lng + (sign2 * outerPoint)}
			select {
			case rgChan <- guess:
				// fmt.Printf("\nInitial Guess: %v \n", guess)
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
				fmt.Printf("Initial Guess: %v\n", guess)
			default:
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()
	return gChan
}
