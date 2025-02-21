package signal

import (
	"math"
	"math/rand"
	"strconv"

	"github.com/davidkleiven/gononlin/nonlin"

	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
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

func GetCQI(sinr float64) int {
	if sinr < CQItoSINRmap[0] {
		return 1
	}

	if sinr > CQItoSINRmap[15] {
		return 15
	}

	for cqi, lowerBound := range CQItoSINRmap {
		upperBound := CQItoSINRmap[cqi+1]
		if sinr >= lowerBound && sinr < upperBound {
			return cqi + 1
		}
	}

	return -1
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

func Sinr(coord model.Coordinate, ueHeight float64, sCell *model.Cell, beamID model.BeamID, interferingBeams []model.BeamID, neighborCells map[types.NCGI]*model.Cell) float64 {
	if math.IsNaN(coord.Lat) || math.IsNaN(coord.Lng) {
		return math.Inf(-1)
	}

	carrier := sCell.GetCarrier(beamID)
	bandwidthHz := bw.MHzToHz(float64(carrier.BsChannelBwDL))
	utils.If(bandwidthHz == 0, 50e6, bandwidthHz)

	noise := calculateNoisePower(bandwidthHz, types.CellType_MACRO)

	mpf := RiceanFading(GetRiceanK(carrier))
	rsrpServing := Strength(coord, ueHeight, mpf, sCell, beamID)
	if rsrpServing == math.Inf(-1) {
		return math.Inf(-1)
	}

	rsrpNeighSum := 0.0
	for _, beamID := range interferingBeams {
		nCell := neighborCells[beamID.NCGI]
		nCarrier := nCell.GetCarrier(beamID)

		mpf := RiceanFading(GetRiceanK(nCarrier))
		nRsrp := Strength(coord, ueHeight, mpf, nCell, beamID)
		if nRsrp == math.Inf(-1) {
			continue
		}
		rsrpNeighSum += nRsrp

	}

	return calculateSinr(rsrpServing, rsrpNeighSum, noise)
}

func SinrF(ueHeight float64, cell *model.Cell, beamID model.BeamID, refSinr float64, neighborBeams []model.BeamID, neighborCells map[types.NCGI]*model.Cell) (f func(out, x []float64)) {

	return func(out, x []float64) {
		coord := model.Coordinate{Lat: x[0], Lng: x[1]}
		fValue := Sinr(coord, ueHeight, cell, beamID, neighborBeams, neighborCells) - refSinr
		out[0] = fValue
		out[1] = fValue
	}
}

func GetSinrPoints(cell *model.Cell, beamID model.BeamID, nCells map[types.NCGI]*model.Cell, nBeamIDs []model.BeamID, ueHeight, refSinr, dc float64, numUes, cqi int) []model.Coordinate {
	sinrPoints := []model.Coordinate{}
	if numUes <= 0 {
		return sinrPoints
	}

	cfp := func(x0 []float64) (f func(out, x []float64)) {
		return SinrF(ueHeight, cell, beamID, refSinr, nBeamIDs, nCells)
	}

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

		sinrPointsCh := ComputePoints(cfp, GetRandGuessesChanUEs(cell, beamID, numUes*overSampling, cqi, 25), newtonKrylovSolver, &stop)
		for sp := range sinrPointsCh {
			if IsPointInsideBoundingBox(sp, cell.BoundingBoxes[beamID]) {
				sinrPoints = append(sinrPoints, sp)
				if len(sinrPoints) >= numUes {
					stop = true
					break SINR_POINTS_LOOP
				}
			}
		}
	}

	return sinrPoints
}

func GetNeighborBeamIDs(neighborCells map[types.NCGI]*model.Cell) []model.BeamID {
	nBeamIDs := []model.BeamID{}
	for nNCGI, nCell := range neighborCells {
		for nCarrierIndex, nCarrier := range nCell.Carriers {
			nCarIndex := nCarrierIndex + 1
			for nBeamIndex := range nCarrier.Beams {
				nBmIndex := nBeamIndex + 1
				nBeamID := model.BeamID{NCGI: nNCGI, CarrierIndex: nCarIndex, BeamIndex: nBmIndex}
				nBeamIDs = append(nBeamIDs, nBeamID)
			}
		}
	}
	return nBeamIDs
}

func GetInterferingBeams(point model.Coordinate, sCell *model.Cell, beamID model.BeamID, cells map[string]*model.Cell) ([]model.BeamID, map[types.NCGI]*model.Cell) {

	interferingBeamIDs := []model.BeamID{}
	interferingCells := map[types.NCGI]*model.Cell{}

	for _, nNCGI := range sCell.Neighbors {
		nCell, exists := cells[strconv.FormatUint(uint64(nNCGI), 10)]
		if !exists {
			continue
		}

		for nCarrierIndex, nCarrier := range nCell.Carriers {
			if nCarrier.ArfcnDL != sCell.GetCarrier(beamID).ArfcnDL {
				continue
			}
			nCarIndex := nCarrierIndex + 1
			for nBeamIndex := range nCarrier.Beams {
				nBmIndex := nBeamIndex + 1
				interferingBeamID := model.BeamID{NCGI: nNCGI, CarrierIndex: nCarIndex, BeamIndex: nBmIndex}
				if IsPointInsideBoundingBox(point, nCell.BoundingBoxes[interferingBeamID]) {
					interferingCells[nNCGI] = nCell
					interferingBeamIDs = append(interferingBeamIDs, interferingBeamID)
				}
			}
		}
	}

	return interferingBeamIDs, interferingCells
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
