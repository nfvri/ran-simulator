package signal

import (
	"math"
	"math/rand"
	"strconv"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"

	mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"

	"github.com/nfvri/ran-simulator/pkg/store/ues"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	log "github.com/sirupsen/logrus"
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
	log.Infof("CQI: %d -- sinr: %f", cqi, sinr)
	return sinr
}

func CalculateUEsLocations(ncgi uint64, numUes int, sinr float64, simModel *model.Model) []model.Coordinate {

	ueHeight := 1.5

	cell := utils.GetCell(types.NCGI(ncgi), simModel)

	neighborCells := []*model.Cell{}
	for _, ncgi := range cell.Neighbors {
		neighborCells = append(neighborCells, utils.GetCell(ncgi, simModel))
	}

	ueLocations := GetSinrPoints(ueHeight, cell, neighborCells, sinr, numUes)

	log.Infof("ueLocations: %v", ueLocations)

	return ueLocations
}

func CreateSimulationUE(ncgi uint64, counter int, sinr float64, location model.Coordinate) (*model.UE, string) {

	imsi := types.IMSI(rand.Int63n(ues.MaxIMSI-ues.MinIMSI) + ues.MinIMSI)
	ueIMSI := strconv.FormatUint(uint64(imsi), 10)

	rrcState := mho.Rrcstatus_RRCSTATUS_CONNECTED

	servingCell := &model.UECell{
		ID:   types.GnbID(ncgi),
		NCGI: types.NCGI(ncgi),
		Rsrp: rand.Float64() * 100,
		Sinr: sinr,
	}

	ue := &model.UE{
		IMSI:        imsi,
		AmfUeNgapID: types.AmfUENgapID(1000 + counter),
		Type:        "phone",
		Location:    location,
		Heading:     0,
		Cell:        servingCell,
		CRNTI:       types.CRNTI(90125 + counter),
		Cells:       []*model.UECell{},
		IsAdmitted:  false,
		RrcState:    rrcState,
	}

	return ue, ueIMSI
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

func CalculateNoisePower(bandwidthHz float64, cellType types.CellType) float64 {
	const (
		Temperature = 290.0 // Kelvin
		Boltzmann   = 1.38e-23
	)

	thermalNoisePower := Boltzmann * Temperature * bandwidthHz // noise power in watts
	thermalNoiseDbm := 10 * math.Log10(thermalNoisePower/1e-3) // convert to dBm

	noiseFigureDb := GetNoiseFigure(bandwidthHz, cellType)

	totalNoiseDbm := thermalNoiseDbm + noiseFigureDb
	return totalNoiseDbm
}

// CellType_FEMTO         ---> WI-Fi
// CellType_ENTERPRISE    ---> 3GPP Micro Cell
// CellType_OUTDOOR_SMALL ---> 3GPP Pico Cell
// CellType_MACRO         ---> 3GPP Macro Cell

// GetNoiseFigure calculates the noise figure based on bandwidth and cell type.
func GetNoiseFigure(bandwidthHz float64, cellType types.CellType) float64 {
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
