package ues

import (
	"context"
	"regexp"
	"strconv"
	"strings"
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

const ACTIVE_UES_METRIC = "DRB.MeanActiveUeDl."
const TOTAL_PRBS_DL_METRIC = "RRU.PrbAvailDl"
const TOTAL_PRBS_UL_METRIC = "RRU.PrbAvailUl"
const USED_PRBS_DL_PATTERN = "RRU.PrbUsedDl.([0-9]|1[0-5])"
const USED_PRBS_UL_PATTERN = "RRU.PrbUsedUl.([0-9]|1[0-5])"

const USED_PRBS_DL_METRIC = "RRU.PrbUsedDl"
const USED_PRBS_UL_METRIC = "RRU.PrbUsedUl"

func InitUEs(cellMeasurements []*metrics.Metric, updatedCells map[string]*model.Cell, cacheStore redisLib.Store, snapshotId string, dc, ueHeight float64) (map[string]*model.UE, bool) {

	cellCqiUesMap, cellPrbsMap := CreateCellInfoMaps(cellMeasurements)

	var ueList = map[string]*model.UE{}

	ctx := context.Background()
	ueGroup, err := cacheStore.GetUEGroup(ctx, snapshotId)
	storeInCache := snapshotId != "" && err != nil

	if err != nil {
		uesLocations, uesSINR := GenerateUEsLocationBasedOnCQI(cellCqiUesMap, updatedCells, ueHeight, dc)
		uesRSRP := GetUEsRsrpBasedOnLocation(uesLocations, updatedCells, ueHeight)

		for sCellNCGI, cqiMap := range cellCqiUesMap {
			sCell, ok := updatedCells[strconv.FormatUint(sCellNCGI, 10)]
			if !ok {
				continue
			}

			totalUEs := 0
			for _, numUEs := range cqiMap {
				totalUEs += numUEs
			}
			if totalUEs == 0 {
				log.Warnf("number of generated ues for cell %v is 0", sCellNCGI)
				continue
			}
			bw.InitBWPs(sCell, cellPrbsMap, sCellNCGI, totalUEs)

			bwpPartitions := bw.PartitionBwps(sCell.Bwps, totalUEs, bw.Lognormally)
			for cqi, numUEs := range cqiMap {
				for i := 0; i < numUEs; i++ {
					if len(uesLocations[sCellNCGI][cqi]) <= i {
						log.Error("number of ue locations generated is smaller than the required")
						break
					}
					ueSINR := uesSINR[sCellNCGI][cqi]
					ueRSRP := uesRSRP[sCellNCGI][cqi][i]
					ueLocation := uesLocations[sCellNCGI][cqi][i]
					ueNeighbors := GetUeNeighbors(ueLocation, sCell, updatedCells, ueHeight, cellPrbsMap)
					totalPrbsDl := cellPrbsMap[sCellNCGI][TOTAL_PRBS_DL_METRIC]
					ueRSRQ := signal.RSRQ(ueSINR, totalPrbsDl)

					simUE, ueIMSI := CreateSimulationUE(sCellNCGI, len(ueList)+1, cqi, totalPrbsDl, ueSINR, ueRSRP, ueRSRQ, ueLocation, ueNeighbors)
					simUE.Cell.BwpRefs = bwpPartitions[i]
					ueList[ueIMSI] = simUE
				}
			}
		}
	} else {
		for imsi := range ueGroup {
			ue := ueGroup[imsi]
			ueList[imsi] = &ue
		}
	}

	log.Infof("------------- len(ueList): %d --------------", len(ueList))
	log.Infof("---------------- Updated UEs -----------------")
	return ueList, storeInCache
}

func CreateCellInfoMaps(cellMeasurements []*metrics.Metric) (map[uint64]map[int]int, map[uint64]map[string]int) {
	//cellPrbsMap[NCGI][CQI]
	cellCQIUEsMap := map[uint64]map[int]int{}
	//cellPrbsMap[NCGI][MetricName]
	cellPrbsMap := map[uint64]map[string]int{}
	for _, metric := range cellMeasurements {
		if _, ok := cellPrbsMap[metric.EntityID]; !ok {
			cellPrbsMap[metric.EntityID] = map[string]int{}
		}
		if strings.Contains(metric.Key, ACTIVE_UES_METRIC) {

			cqi, err := strconv.Atoi(metric.Key[len(ACTIVE_UES_METRIC):])
			if err != nil {
				log.Errorf("Error converting CQI level to integer: %v", err)
				continue
			}

			if _, exists := cellCQIUEsMap[metric.EntityID]; !exists {
				cellCQIUEsMap[metric.EntityID] = make(map[int]int)
			}
			numUEs, _ := strconv.Atoi(metric.GetValue())

			// Metrics in the list are ordered chronologically
			// from oldest at the beginning to newest at the end.
			// Keep the latest metric
			cellCQIUEsMap[metric.EntityID][cqi] = numUEs
		}
		if metric.Key == TOTAL_PRBS_DL_METRIC {
			totalPrbsDl, _ := strconv.Atoi(metric.GetValue())
			cellPrbsMap[metric.EntityID][TOTAL_PRBS_DL_METRIC] = totalPrbsDl
		}
		if metric.Key == TOTAL_PRBS_UL_METRIC {
			totalPrbsUl, _ := strconv.Atoi(metric.GetValue())
			cellPrbsMap[metric.EntityID][TOTAL_PRBS_UL_METRIC] = totalPrbsUl
		}
		if matchesPattern(metric.Key, USED_PRBS_DL_PATTERN) {
			usedPrbsDl, _ := strconv.Atoi(metric.GetValue())
			cellPrbsMap[metric.EntityID][USED_PRBS_DL_METRIC] += usedPrbsDl
		}
		if matchesPattern(metric.Key, USED_PRBS_UL_PATTERN) {
			usedPrbsUl, _ := strconv.Atoi(metric.GetValue())
			cellPrbsMap[metric.EntityID][USED_PRBS_UL_METRIC] += usedPrbsUl
		}
	}
	return cellCQIUEsMap, cellPrbsMap
}

func matchesPattern(metric, p string) bool {
	r, err := regexp.Compile(p)
	if err != nil {
		return false
	}
	return r.MatchString(metric)
}

func GenerateUEsLocationBasedOnCQI(cellCqiUesMap map[uint64]map[int]int, simModelCells map[string]*model.Cell, ueHeight, dc float64) (map[uint64]map[int][]model.Coordinate, map[uint64]map[int]float64) {
	// map[servingCellNCGI]map[CQI][]Locations
	uesLocations := make(map[uint64]map[int][]model.Coordinate)

	// map[servingCellNCGI]map[CQI]SINR
	uesSINR := make(map[uint64]map[int]float64)

	var wg sync.WaitGroup

	for sCellNCGI, cqiMap := range cellCqiUesMap {

		if _, exists := uesSINR[sCellNCGI]; !exists {
			uesSINR[sCellNCGI] = make(map[int]float64)
		}
		if _, exists := uesLocations[sCellNCGI]; !exists {
			uesLocations[sCellNCGI] = make(map[int][]model.Coordinate)
		}

		for cqi, numUEs := range cqiMap {
			wg.Add(1)
			go func(sCellNCGI uint64, cqi, numUEs int) {
				defer wg.Done()
				ueSINR := signal.GetSINR(cqi)

				ueLocationForCqi := signal.GenerateUEsLocations(sCellNCGI, numUEs, cqi, ueSINR, ueHeight, dc, simModelCells)
				uesLocations[sCellNCGI][cqi] = ueLocationForCqi
				uesSINR[sCellNCGI][cqi] = ueSINR
			}(sCellNCGI, cqi, numUEs)
		}

	}
	wg.Wait()

	return uesLocations, uesSINR
}

func GetUEsRsrpBasedOnLocation(uesLocations map[uint64]map[int][]model.Coordinate, simModelCells map[string]*model.Cell, ueHeight float64) map[uint64]map[int][]float64 {

	// map[servingCellNCGI]map[CQI]RSRP
	uesRSRP := make(map[uint64]map[int][]float64)

	for sCellNCGI, cqiMap := range uesLocations {
		if _, exists := uesRSRP[sCellNCGI]; !exists {
			uesRSRP[sCellNCGI] = make(map[int][]float64)
		}
		sCell, ok := simModelCells[strconv.FormatUint(sCellNCGI, 10)]
		if !ok {
			continue
		}

		mpf := signal.RiceanFading(signal.GetRiceanK(sCell))

		for cqi, cellUesLocations := range cqiMap {
			uesRSRP[sCellNCGI][cqi] = make([]float64, len(uesLocations[sCellNCGI][cqi]))
			for i, ueCoord := range cellUesLocations {
				uesRSRP[sCellNCGI][cqi][i] = signal.Strength(ueCoord, ueHeight, mpf, sCell)
			}

		}

	}
	return uesRSRP
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

func GetUeNeighbors(point model.Coordinate, sCell *model.Cell, simModelCells map[string]*model.Cell, ueHeight float64, cellPrbsMap map[uint64]map[string]int) []*model.UECell {
	ueNeighbors := []*model.UECell{}

	neighborCells := utils.GetNeighborCells(sCell, simModelCells)
	for _, nCell := range neighborCells {
		if signal.IsPointInsideBoundingBox(point, nCell.BoundingBox) {
			mpf := signal.RiceanFading(signal.GetRiceanK(nCell))
			nCellNeigh := utils.GetNeighborCells(nCell, simModelCells)
			rsrp := signal.Strength(point, ueHeight, mpf, nCell)
			sinr := signal.Sinr(point, ueHeight, nCell, nCellNeigh)
			rsrq := signal.RSRQ(sinr, 24)

			ueCell := &model.UECell{
				ID:          types.GnbID(nCell.NCGI),
				NCGI:        nCell.NCGI,
				Rsrp:        rsrp,
				Rsrq:        rsrq,
				Sinr:        sinr,
				TotalPrbsDl: cellPrbsMap[uint64(nCell.NCGI)][TOTAL_PRBS_DL_METRIC],
			}
			ueNeighbors = append(ueNeighbors, ueCell)
		}
	}
	return ueNeighbors
}
