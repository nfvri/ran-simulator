package ues

import (
	"context"
	"math"
	"strconv"
	"sync"

	"github.com/nfvri/onos-api/go/onos/ransim/metrics"
	"github.com/nfvri/onos-api/go/onos/ransim/types"
	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/signal"
	redisLib "github.com/nfvri/ran-simulator/pkg/store/redis"
	"github.com/nfvri/ran-simulator/pkg/utils"
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
	ues := map[string]*model.UE{}
	cellServedUEs := []*model.UE{}
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

		numUEsPerBeamQS := bw.GetNumUEsPerBeamQS(sCell, numUEsPerCQI)

		totalUEs := 0
		for _, numUEs := range numUEsPerBeamQS {
			totalUEs += numUEs
		}
		if totalUEs == 0 {
			log.Warnf("number of generated ues for cell %v is 0", sCellNCGI)
			continue
		}

		ues, cellServedUEs = GenerateUEsBasedOnBeamQS(sCell, cells, numUEsPerBeamQS, ueHeight, dc, prbMeasPerCell, ues)

		usedPRBsDL := usedPRBsDLPerCQIByCell[sCellNCGI]
		usedPRBsUL := usedPRBsULPerCQIByCell[sCellNCGI]
		availPRBsDL := prbMeasPerCell[sCellNCGI][bw.AVAIL_PRBS_DL_METRIC]
		availPRBsUL := prbMeasPerCell[sCellNCGI][bw.AVAIL_PRBS_UL_METRIC]

		bw.InitBWPs(sCell, numUEsPerCQI, usedPRBsDL, usedPRBsUL, availPRBsDL, availPRBsUL, cellServedUEs)
	}

	log.Infof("------------- len(ues): %d --------------", len(ues))
	log.Infof("---------------- Updated UEs -----------------")
	return ues, storeInCache
}

func GenerateUEsBasedOnBeamQS(sCell *model.Cell, cells map[string]*model.Cell, numUEsPerBeamQS map[model.BeamQS]int, ueHeight float64, dc float64, prbMeasPerCell map[uint64]map[string]int, ues map[string]*model.UE) (map[string]*model.UE, []*model.UE) {
	nCells := utils.GetNeighborCells(sCell, cells)
	nBeamIDs := signal.GetNeighborBeamIDs(nCells)

	cellServedUEs := []*model.UE{}
	mtx := sync.RWMutex{}
	var wg sync.WaitGroup

	for beamQS, numUEs := range numUEsPerBeamQS {
		if numUEs <= 0 {
			continue
		}
		wg.Add(1)
		go func(servCell *model.Cell, beamQs model.BeamQS, numUes int) {
			defer wg.Done()
			ueSINR := signal.GetSINR(beamQs.CQI)
			ueLocations := signal.GetSinrPoints(servCell, beamQs.BeamID, nCells, nBeamIDs, ueHeight, ueSINR, dc, numUes, beamQs.CQI)
			ueRSRPs := GetUERsrpsBasedOnLocation(servCell, beamQs.BeamID, ueLocations, cells, ueHeight)

			for i := 0; i < numUes; i++ {
				if len(ueLocations) <= i {
					log.Error("number of ue locations generated is smaller than the required")
					break
				}

				ueRSRP := ueRSRPs[i]
				ueLocation := ueLocations[i]
				ueNeighbors := InitUeNeighbors(ueLocation, servCell, beamQs.BeamID, cells, ueHeight, prbMeasPerCell)
				totalPrbsDl := prbMeasPerCell[uint64(sCell.NCGI)][bw.AVAIL_PRBS_DL_METRIC]
				ueRSRQ := math.Round(signal.RSRQ(ueSINR, totalPrbsDl)*100) / 100

				mtx.Lock()
				counter := len(ues) + 1
				mtx.Unlock()
				simUE, ueIMSI := CreateSimulationUE(uint64(sCell.NCGI), beamQs, counter, totalPrbsDl, ueHeight, ueSINR, ueRSRP, ueRSRQ, ueLocation, ueNeighbors)

				mtx.Lock()
				ues[ueIMSI] = simUE
				cellServedUEs = append(cellServedUEs, simUE)
				mtx.Unlock()
			}
		}(sCell, beamQS, numUEs)
	}
	wg.Wait()
	return ues, cellServedUEs
}

func GetUERsrpsBasedOnLocation(sCell *model.Cell, beamID model.BeamID, uesLocations []model.Coordinate, cells map[string]*model.Cell, ueHeight float64) (ueRSRPs []float64) {

	ueRSRPs = []float64{}
	mpf := signal.RiceanFading(signal.GetRiceanK(sCell.GetCarrier(beamID)))

	for _, ueCoord := range uesLocations {
		rsrp := signal.Strength(ueCoord, ueHeight, mpf, sCell, beamID)
		ueRSRPs = append(ueRSRPs, math.Round(rsrp*100)/100)
	}

	return
}

func CreateSimulationUE(ncgi uint64, beamQS model.BeamQS, counter, totalPrbsDl int, ueHeight, sinr, rsrp, rsrq float64, location model.Coordinate, neighborCells []*model.UECell) (*model.UE, string) {

	imsi := utils.ImsiGenerator(counter)
	ueIMSI := strconv.FormatUint(uint64(imsi), 10)

	rrcState := mho.Rrcstatus_RRCSTATUS_CONNECTED
	// add neighbours
	servingCell := &model.UECell{
		ID:          types.GnbID(ncgi),
		NCGI:        types.NCGI(ncgi),
		BeamID:      model.BeamID{NCGI: beamQS.BeamID.NCGI, CarrierIndex: beamQS.BeamID.CarrierIndex, BeamIndex: beamQS.BeamID.BeamIndex},
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
		FiveQi:      beamQS.CQI,
		CRNTI:       types.CRNTI(90125 + counter),
		Cells:       neighborCells,
		IsAdmitted:  false,
		Height:      ueHeight,
		RrcState:    rrcState,
	}

	return ue, ueIMSI
}

func InitUeNeighbors(point model.Coordinate, sCell *model.Cell, beamID model.BeamID, cells map[string]*model.Cell, ueHeight float64, prbMeasPerCell map[uint64]map[string]int) []*model.UECell {
	ueNeighbors := []*model.UECell{}

	interferingBeamIDs, neighborCells := signal.GetInterferingBeams(point, sCell, beamID, cells)

	for _, nBeamID := range interferingBeamIDs {
		nCell, ok := neighborCells[nBeamID.NCGI]
		if !ok {
			continue
		}
		nCarrier := nCell.GetCarrier(nBeamID)

		if signal.IsPointInsideBoundingBox(point, nCell.BoundingBoxes[nBeamID]) {

			mpf := signal.RiceanFading(signal.GetRiceanK(nCarrier))
			interfBeamIDs, interfCells := signal.GetInterferingBeams(point, nCell, nBeamID, cells)
			rsrp := signal.Strength(point, ueHeight, mpf, nCell, nBeamID)
			sinr := signal.Sinr(point, ueHeight, nCell, nBeamID, interfBeamIDs, interfCells)
			rsrq := signal.RSRQ(sinr, 24)

			ueCell := &model.UECell{
				ID:          types.GnbID(nBeamID.NCGI),
				NCGI:        nBeamID.NCGI,
				BeamID:      nBeamID,
				Rsrp:        math.Round(rsrp*100) / 100,
				Rsrq:        math.Round(rsrq*100) / 100,
				Sinr:        math.Round(sinr*100) / 100,
				AvailPrbsDl: prbMeasPerCell[uint64(nBeamID.NCGI)][bw.AVAIL_PRBS_DL_METRIC],
			}
			ueNeighbors = append(ueNeighbors, ueCell)
		}
	}

	return ueNeighbors
}
