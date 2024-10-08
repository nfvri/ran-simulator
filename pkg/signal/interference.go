package signal

import (
	"math"
	"math/rand"
	"strconv"

	"github.com/davidkleiven/gononlin/nonlin"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"

	"github.com/onosproject/onos-api/go/onos/ransim/types"
)

// CQItoSINR mapping
// 0 and 16 values included only for calculations, not valid CQI indexes
var CQItoSINRmap = map[int]float64{
	0:  -8.950,
	1:  -6.9360,
	2:  -5.1470,
	3:  -3.1800,
	4:  -1.2530,
	5:  0.7610,
	6:  2.6990,
	7:  4.6940,
	8:  6.5250,
	9:  8.5730,
	10: 10.3660,
	11: 12.2890,
	12: 14.1730,
	13: 15.8880,
	14: 17.8140,
	15: 19.8290,
	16: 21.843,
}

func GetSINR(cqi int) float64 {

	lowerBound := CQItoSINRmap[cqi-1]
	upperBound := CQItoSINRmap[cqi]

	sinr := lowerBound + math.Abs(rand.Float64()*(upperBound-lowerBound))
	return sinr
}
func GenerateUEsLocations(ncgi uint64, numUes, cqi int, sinr, ueHeight, dc float64, simModelCells map[string]*model.Cell) []model.Coordinate {

	cell, ok := simModelCells[strconv.FormatUint(ncgi, 10)]
	if !ok {
		return []model.Coordinate{}
	}
	neighborCells := utils.GetNeighborCells(cell, simModelCells)

	ueLocations := GetSinrPoints(ueHeight, cell, neighborCells, sinr, dc, numUes, cqi)

	return ueLocations
}

func calculateSinr(rsrpServingDbm, rsrpNeighSumDbm, noiseDbm float64) float64 {

	rsrpServingMw := utils.DbmToMw(rsrpServingDbm)
	noiseMw := utils.DbmToMw(noiseDbm)

	sinrMw := rsrpServingMw / noiseMw
	if rsrpNeighSumDbm != 0.0 {
		interferenceMw := utils.DbmToMw(rsrpNeighSumDbm)
		sinrMw = rsrpServingMw / (interferenceMw + noiseMw)
	}

	sinrDbm := utils.MwToDbm(sinrMw)

	return sinrDbm
}

func Sinr(coord model.Coordinate, ueHeight float64, sCell *model.Cell, neighborCells []*model.Cell) float64 {
	if math.IsNaN(coord.Lat) || math.IsNaN(coord.Lng) {
		return math.Inf(-1)
	}

	bandwidthHz := float64(sCell.Channel.BsChannelBwDL) * 1e6
	utils.If(bandwidthHz == 0, 50e6, bandwidthHz)

	noise := calculateNoisePower(bandwidthHz, types.CellType_MACRO)

	mpf := RiceanFading(GetRiceanK(sCell))
	rsrpServing := Strength(coord, ueHeight, mpf, *sCell)
	if rsrpServing == math.Inf(-1) {
		return math.Inf(-1)
	}

	rsrpNeighSum := 0.0
	for _, n := range neighborCells {

		mpf := RiceanFading(GetRiceanK(n))

		nRsrp := Strength(coord, ueHeight, mpf, *n)
		if nRsrp == math.Inf(-1) {
			continue
		}
		rsrpNeighSum += nRsrp
	}

	return calculateSinr(rsrpServing, rsrpNeighSum, noise)
}

func SinrF(ueHeight float64, cell *model.Cell, refSinr float64, neighborCells []*model.Cell) (f func(out, x []float64)) {

	return func(out, x []float64) {
		coord := model.Coordinate{Lat: x[0], Lng: x[1]}
		fValue := Sinr(coord, ueHeight, cell, neighborCells) - refSinr
		out[0] = fValue
		out[1] = fValue
	}
}

func GetSinrPoints(ueHeight float64, cell *model.Cell, neighborCells []*model.Cell, refSinr, dc float64, numUes, cqi int) []model.Coordinate {

	cfp := func(x0 []float64) (f func(out, x []float64)) {
		return SinrF(ueHeight, cell, refSinr, neighborCells)
	}

	sinrPoints := []model.Coordinate{}
	stepSizeMeters := 10.0
	overSampling := 100
	maxIter := 300
	stop := false

	newtonKrylovSolver := nonlin.NewtonKrylov{
		Maxiter:  maxIter,
		StepSize: utils.MetersToLatDegrees(stepSizeMeters),
		Tol:      0.5,
	}

SINR_POINTS_LOOP:
	for {

		sinrPointsCh := ComputePoints(cfp, GetRandGuessesChanUEs(*cell, numUes*overSampling, cqi, 25), newtonKrylovSolver, &stop)
		for sp := range sinrPointsCh {
			if IsPointInsideBoundingBox(sp, cell.BoundingBox) {
				sinrPoints = append(sinrPoints, sp)
				if len(sinrPoints) >= numUes {
					stop = true
					break SINR_POINTS_LOOP
				}
			}
		}
	}

	return utils.SortCoordinatesByBearing(cell.Sector.Center, sinrPoints)
}

func calculateNoisePower(bandwidthHz float64, cellType types.CellType) float64 {
	const (
		Temperature = 290.0 // Kelvin
		Boltzmann   = 1.38e-23
	)

	thermalNoisePower := Boltzmann * Temperature * bandwidthHz // noise power in watts
	thermalNoiseDbm := utils.MwToDbm(thermalNoisePower / 1e-3) // convert to dBm

	noiseFigureDbm := getNoiseFigure(bandwidthHz, cellType)

	totalNoiseDbm := thermalNoiseDbm + noiseFigureDbm
	return totalNoiseDbm
}

// CellType_FEMTO         ---> WI-Fi
// CellType_ENTERPRISE    ---> 3GPP Micro Cell
// CellType_OUTDOOR_SMALL ---> 3GPP Pico Cell
// CellType_MACRO         ---> 3GPP Macro Cell

// getNoiseFigure calculates the noise figure based on bandwidth and cell type.
func getNoiseFigure(bandwidthHz float64, cellType types.CellType) float64 {
	var NF float64

	// Determine base noise figure based on bandwidth
	switch {
	case bandwidthHz >= 20e6:
		NF = 9.0
	case bandwidthHz >= 15e6:
		NF = 8.0
	case bandwidthHz >= 10e6:
		NF = 7.0
	default:
		NF = 6.0
	}

	// Adjust noise figure based on cell type
	switch cellType {
	case types.CellType_OUTDOOR_SMALL, types.CellType_FEMTO:
		NF += 8.0
	case types.CellType_ENTERPRISE:
		NF += 5.0
	default:
		// no-op
	}

	return NF
}
