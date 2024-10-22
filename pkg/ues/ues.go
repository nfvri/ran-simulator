package ues

import (
	"context"
	"strconv"
	"sync"

	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/signal"
	redisLib "github.com/nfvri/ran-simulator/pkg/store/redis"
	"github.com/nfvri/ran-simulator/pkg/utils"
	"github.com/onosproject/onos-api/go/onos/ransim/metrics"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"
	log "github.com/sirupsen/logrus"
)

func InitUEs(cellMeasurements []*metrics.Metric, cells map[string]*model.Cell, cacheStore redisLib.Store, snapshotId string, dc, ueHeight float64) (map[string]*model.UE, bool) {

	numUEsByCell, prbMeasPerCell := bw.UtilizationInfoByCell(cellMeasurements)
	numUEsPerCQIByCell := bw.GetNumUEsPerCQIByCell(numUEsByCell)
	usedPRBsDLPerCQIByCell, usedPRBsULPerCQIByCell := bw.GetUsedPRBsPerCQIByCell(prbMeasPerCell, numUEsPerCQIByCell)

	for sCellNCGI, numUEsPerCQI := range numUEsPerCQIByCell {
		log.Infof("Cell: %v -- numUEsPerCQI: %v\n\n", sCellNCGI, numUEsPerCQI)
	}
	for sCellNCGI, prbMeas := range prbMeasPerCell {
		log.Infof("Cell: %v -- prbMeas: %v\n\n", sCellNCGI, prbMeas)
	}

	for sCellNCGI, usedPRBsDL := range usedPRBsDLPerCQIByCell {
		sum := 0
		for _, numPRBs := range usedPRBsDL {
			sum += numPRBs
		}
		log.Infof("Cell: %v -- usedPrbsDl: %v", sCellNCGI, sum)
	}
	for sCellNCGI, usedPRBsUL := range usedPRBsULPerCQIByCell {
		sum := 0
		for _, numPRBs := range usedPRBsUL {
			sum += numPRBs
		}
		log.Infof("Cell: %v -- usedPrbsUl: %v", sCellNCGI, sum)
	}

	var ues = map[string]*model.UE{}
	ctx := context.Background()
	ueGroup, err := cacheStore.GetUEGroup(ctx, snapshotId)
	storeInCache := snapshotId != "" && err != nil

	if err == nil {
		for imsi := range ueGroup {
			ue := ueGroup[imsi]
			ues[imsi] = &ue
		}
		return ues, storeInCache
	}

	for sCellNCGI, numUEsPerCQI := range numUEsPerCQIByCell {

		sCell, ok := cells[strconv.FormatUint(sCellNCGI, 10)]
		if !ok {
			continue
		}

		ueLocationsPerCQI := GenerateUELocationsBasedOnCQI(sCell, numUEsPerCQI, cells, ueHeight, dc)
		ueRSRPsPerCQI := GetUERsrpsBasedOnLocation(sCell, ueLocationsPerCQI, cells, ueHeight)

		totalUEs := 0
		for _, numUEs := range numUEsPerCQI {
			totalUEs += numUEs
		}
		if totalUEs == 0 {
			log.Warnf("number of generated ues for cell %v is 0", sCellNCGI)
			continue
		}

		cellServedUEs := []*model.UE{}
		statsPerCQI := map[int]bw.CQIStats{}
		for cqi, numUEs := range numUEsPerCQI {
			ueSINR := signal.GetSINR(cqi)
			for i := 0; i < numUEs; i++ {
				if len(ueLocationsPerCQI[cqi]) <= i {
					log.Error("number of ue locations generated is smaller than the required")
					break
				}

				ueRSRP := ueRSRPsPerCQI[cqi][i]
				ueLocation := ueLocationsPerCQI[cqi][i]
				ueNeighbors := InitUeNeighbors(ueLocation, sCell, cells, ueHeight, prbMeasPerCell)
				totalPrbsDl := prbMeasPerCell[sCellNCGI][bw.AVAIL_PRBS_DL_METRIC]
				ueRSRQ := signal.RSRQ(ueSINR, totalPrbsDl)

				simUE, ueIMSI := CreateSimulationUE(sCellNCGI, len(ues)+1, cqi, totalPrbsDl, ueSINR, ueRSRP, ueRSRQ, ueLocation, ueNeighbors)
				ues[ueIMSI] = simUE
				cellServedUEs = append(cellServedUEs, simUE)
			}

			if numUEs > 0 {
				statsPerCQI[cqi] = bw.CQIStats{
					NumUEs:     numUEs,
					UsedPRBsDL: usedPRBsDLPerCQIByCell[sCellNCGI][cqi],
					UsedPRBsUL: usedPRBsULPerCQIByCell[sCellNCGI][cqi],
				}
			}
		}
		availPRBsDL := prbMeasPerCell[sCellNCGI][bw.AVAIL_PRBS_DL_METRIC]
		availPRBsUL := prbMeasPerCell[sCellNCGI][bw.AVAIL_PRBS_UL_METRIC]

		bw.InitBWPs(sCell, statsPerCQI, availPRBsDL, availPRBsUL, cellServedUEs)
	}

	log.Infof("------------- len(ues): %d --------------", len(ues))
	log.Infof("---------------- Updated UEs -----------------")
	return ues, storeInCache
}

func GenerateUELocationsBasedOnCQI(sCell *model.Cell, numUesPerCQI map[int]int, cells map[string]*model.Cell, ueHeight, dc float64) (uesLocationsPerCQI map[int][]model.Coordinate) {

	uesLocationsPerCQI = make(map[int][]model.Coordinate, 15)
	mtx := sync.RWMutex{}
	var wg sync.WaitGroup
	for cqi, numUEs := range numUesPerCQI {
		wg.Add(1)
		go func(CQI, numUEs int) {
			defer wg.Done()
			ueSINR := signal.GetSINR(CQI)
			neighborCells := utils.GetNeighborCells(sCell, cells)
			mtx.Lock()
			uesLocationsPerCQI[CQI] = signal.GetSinrPoints(ueHeight, sCell, neighborCells, ueSINR, dc, numUEs, CQI)
			mtx.Unlock()
		}(cqi, numUEs)
	}
	wg.Wait()

	return
}

func GetUERsrpsBasedOnLocation(sCell *model.Cell, uesLocationsPerCQI map[int][]model.Coordinate, cells map[string]*model.Cell, ueHeight float64) (uesRSRPPerCQI map[int][]float64) {

	uesRSRPPerCQI = make(map[int][]float64)
	mpf := signal.RiceanFading(signal.GetRiceanK(sCell))

	for cqi, uesLocations := range uesLocationsPerCQI {
		uesRSRPPerCQI[cqi] = make([]float64, len(uesLocationsPerCQI[cqi]))
		for index, ueCoord := range uesLocations {
			uesRSRPPerCQI[cqi][index] = signal.Strength(ueCoord, ueHeight, mpf, sCell)
		}
	}

	return
}

func CreateSimulationUE(ncgi uint64, counter, cqi, totalPrbsDl int, sinr, rsrp, rsrq float64, location model.Coordinate, neighborCells []*model.UECell) (*model.UE, string) {

	imsi := utils.ImsiGenerator(counter)
	ueIMSI := strconv.FormatUint(uint64(imsi), 10)

	rrcState := mho.Rrcstatus_RRCSTATUS_CONNECTED
	// add neighbours
	servingCell := &model.UECell{
		ID:          types.GnbID(ncgi),
		NCGI:        types.NCGI(ncgi),
		Rsrq:        rsrq,
		Rsrp:        rsrp,
		Sinr:        sinr,
		AvailPrbsDl: totalPrbsDl,
	}

	ue := &model.UE{
		IMSI:        imsi,
		AmfUeNgapID: types.AmfUENgapID(1000 + counter),
		Type:        "phone",
		Location:    location,
		Heading:     0,
		Cell:        servingCell,
		FiveQi:      cqi,
		CRNTI:       types.CRNTI(90125 + counter),
		Cells:       neighborCells,
		IsAdmitted:  false,
		RrcState:    rrcState,
	}

	return ue, ueIMSI
}

func InitUeNeighbors(point model.Coordinate, sCell *model.Cell, cells map[string]*model.Cell, ueHeight float64, prbMeasPerCell map[uint64]map[string]int) []*model.UECell {
	ueNeighbors := []*model.UECell{}

	neighborCells := utils.GetNeighborCells(sCell, cells)
	for _, nCell := range neighborCells {
		if signal.IsPointInsideBoundingBox(point, nCell.BoundingBox) {
			mpf := signal.RiceanFading(signal.GetRiceanK(nCell))
			nCellNeigh := utils.GetNeighborCells(nCell, cells)
			rsrp := signal.Strength(point, ueHeight, mpf, nCell)
			sinr := signal.Sinr(point, ueHeight, nCell, nCellNeigh)
			rsrq := signal.RSRQ(sinr, 24)

			ueCell := &model.UECell{
				ID:          types.GnbID(nCell.NCGI),
				NCGI:        nCell.NCGI,
				Rsrp:        rsrp,
				Rsrq:        rsrq,
				Sinr:        sinr,
				AvailPrbsDl: prbMeasPerCell[uint64(nCell.NCGI)][bw.AVAIL_PRBS_DL_METRIC],
			}
			ueNeighbors = append(ueNeighbors, ueCell)
		}
	}
	return ueNeighbors
}
