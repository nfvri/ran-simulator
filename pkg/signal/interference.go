package signal

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
)

func Sinr(coord model.Coordinate, ueHeight float64, sCell *model.Cell, neighborCells []*model.Cell) float64 {
	bandwidth := 20e6 // 20 MHz bandwidth
	noiseFigure := 5.0
	thermalNoise := thermalNoisePower(bandwidth)
	noiseFigureLinear := math.Pow(10, noiseFigure/10)
	totalNoisePower := thermalNoise * noiseFigureLinear
	noise := rand.NormFloat64() * math.Sqrt(totalNoisePower)
	// log.Info(noise)
	scaleNu := 1.0
	scaleSigma := 0.09
	K := rand.NormFloat64()*RICEAN_K_STD_MACRO + RICEAN_K_MEAN
	mpf := RiceanFading(K, scaleNu, scaleSigma)
	rsrpServing := Strength(coord, ueHeight, mpf, *sCell)
	fmt.Printf("\nrsprServing: %v\n", rsrpServing)
	rsrpNeighSum := 0.0

	for _, n := range neighborCells {
		mpf := RiceanFading(K, scaleNu, scaleSigma)
		rsrpNeighSum += Strength(coord, ueHeight, mpf, *n)
		fmt.Printf("\nrsrpNeighSum: %v\n", rsrpNeighSum)
	}
	return rsrpServing / (rsrpNeighSum + noise)
}

func SinrF(ueHeight float64, cell *model.Cell, refSinr float64, neighborCells []*model.Cell) (f func(out, x []float64)) {

	return func(out, x []float64) {
		coord := model.Coordinate{Lat: x[0], Lng: x[1]}
		out[0] = Sinr(coord, ueHeight, cell, neighborCells) - refSinr
		out[1] = Sinr(coord, ueHeight, cell, neighborCells) - refSinr
	}
}

func GetSinrPoints(ueHeight float64, cell *model.Cell, neighborCells []*model.Cell, refSinr float64, numUes int) []model.Coordinate {

	cfp := func(x0 []float64) (f func(out, x []float64)) {
		return SinrF(ueHeight, cell, refSinr, neighborCells)
	}

	var sinrPoints []model.Coordinate
	for i := 1; i <= 100; i += 10 {
		sinrPoints = []model.Coordinate{}
		sinrPointsCh := ComputePointsWithNewtonKrylov(cfp, GetRandGuessesChan(*cell, numUes*i), 100)
		for sp := range sinrPointsCh {
			sinrPoints = append(sinrPoints, sp)
		}
		if len(sinrPoints) >= numUes {
			break
		}
	}
	return utils.SortCoordinatesByBearing(cell.Sector.Center, sinrPoints)
}
