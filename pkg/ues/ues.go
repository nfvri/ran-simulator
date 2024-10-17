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

	numUEsPerCQIByCell, prbMeasPerCell := bw.CreateCellInfoMaps(cellMeasurements)

	var ueList = map[string]*model.UE{}
	ctx := context.Background()
	ueGroup, err := cacheStore.GetUEGroup(ctx, snapshotId)
	storeInCache := snapshotId != "" && err != nil

	if err == nil {
		for imsi := range ueGroup {
			ue := ueGroup[imsi]
			ueList[imsi] = &ue
		}
		return ueList, storeInCache
	}

	for sCellNCGI, numUEsPerCQI := range numUEsPerCQIByCell {

		sCell, ok := cells[strconv.FormatUint(sCellNCGI, 10)]
		if !ok {
			continue
		}

		uesLocationsPerCQI := GenerateUEsLocationBasedOnCQI(sCell, numUEsPerCQIByCell[sCellNCGI], cells, ueHeight, dc)
		uesRSRPPerCQI := GetUEsRsrpBasedOnLocation(sCell, uesLocationsPerCQI, cells, ueHeight)

		totalUEs := 0
		for _, numUEs := range numUEsPerCQI {
			totalUEs += numUEs
		}
		if totalUEs == 0 {
			log.Warnf("number of generated ues for cell %v is 0", sCellNCGI)
			continue
		}

		cellServedUEs := []*model.UE{}
		for cqi, numUEs := range numUEsPerCQI {
			ueSINR := signal.GetSINR(cqi)
			for i := 0; i < numUEs; i++ {
				if len(uesLocationsPerCQI[cqi]) <= i {
					log.Error("number of ue locations generated is smaller than the required")
					break
				}

				ueRSRP := uesRSRPPerCQI[cqi][i]
				ueLocation := uesLocationsPerCQI[cqi][i]
				ueNeighbors := InitUeNeighbors(ueLocation, sCell, cells, ueHeight, prbMeasPerCell)
				totalPrbsDl := prbMeasPerCell[sCellNCGI][bw.TOTAL_PRBS_DL_METRIC]
				ueRSRQ := signal.RSRQ(ueSINR, totalPrbsDl)

				simUE, ueIMSI := CreateSimulationUE(sCellNCGI, len(ueList)+1, cqi, totalPrbsDl, ueSINR, ueRSRP, ueRSRQ, ueLocation, ueNeighbors)
				ueList[ueIMSI] = simUE
				cellServedUEs = append(cellServedUEs, simUE)
			}
		}
		bw.InitBWPs(sCell, prbMeasPerCell[sCellNCGI], cellServedUEs)
	}

	log.Infof("------------- len(ueList): %d --------------", len(ueList))
	log.Infof("---------------- Updated UEs -----------------")
	return ueList, storeInCache
}

func GenerateUEsLocationBasedOnCQI(sCell *model.Cell, numUesPerCQI map[int]int, cells map[string]*model.Cell, ueHeight, dc float64) (uesLocationsPerCQI map[int][]model.Coordinate) {

	uesSINR := make(map[int]float64)
	uesLocationsPerCQI = make(map[int][]model.Coordinate)

	var wg sync.WaitGroup
	for cqi, numUEs := range numUesPerCQI {
		wg.Add(1)
		go func(cqi, numUEs int) {
			defer wg.Done()
			ueSINR := signal.GetSINR(cqi)
			neighborCells := utils.GetNeighborCells(sCell, cells)
			uesLocationsPerCQI[cqi] = signal.GetSinrPoints(ueHeight, sCell, neighborCells, ueSINR, dc, numUEs, cqi)
			uesSINR[cqi] = ueSINR
		}(cqi, numUEs)
	}
	wg.Wait()

	return
}

func GetUEsRsrpBasedOnLocation(sCell *model.Cell, uesLocationsPerCQI map[int][]model.Coordinate, cells map[string]*model.Cell, ueHeight float64) (uesRSRPPerCQI map[int][]float64) {

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
		TotalPrbsDl: totalPrbsDl,
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

func InitUeNeighbors(point model.Coordinate, sCell *model.Cell, cells map[string]*model.Cell, ueHeight float64, cellPrbsMap map[uint64]map[string]int) []*model.UECell {
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
				TotalPrbsDl: cellPrbsMap[uint64(nCell.NCGI)][bw.TOTAL_PRBS_DL_METRIC],
			}
			ueNeighbors = append(ueNeighbors, ueCell)
		}
	}
	return ueNeighbors
}
