package ues

import (
	"context"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"

	model_ransim "github.com/nfvri/ran-simulator/pkg/model"
	signal_ransim "github.com/nfvri/ran-simulator/pkg/signal"
	redis_ransim "github.com/nfvri/ran-simulator/pkg/store/redis"
	metrics_ransim "github.com/onosproject/onos-api/go/onos/ransim/metrics"
	log "github.com/sirupsen/logrus"
)

const ACTIVE_UES_METRIC = "DRB.MeanActiveUeDl."
const TOTAL_PRBS_DL_METRIC = "RRU.PrbAvailDl"
const TOTAL_PRBS_UL_METRIC = "RRU.PrbAvailUl"
const USED_PRBS_DL_METRIC = "RRU.PrbUsedDl.([0-9]|1[0-5])"
const USED_PRBS_UL_METRIC = "RRU.PrbUsedUl.([0-9]|1[0-5])"

func InitUEs(cellMeasurements []*metrics_ransim.Metric, updatedCells map[string]*model_ransim.Cell, cacheStore redis_ransim.Store, snapshotId string, dc, ueHeight float64) (map[string]model_ransim.UE, bool) {

	cellCqiUesMap, cellPrbsMap := CreateCellInfoMaps(cellMeasurements)

	var ueList map[string]model_ransim.UE

	ctx := context.Background()
	ueGroup, err := cacheStore.GetUEGroup(ctx, snapshotId)
	storeInCache := snapshotId != "" && err != nil

	if err != nil {
		uesLocations, uesSINR := GenerateUEsLocationBasedOnCQI(cellCqiUesMap, updatedCells, ueHeight, dc)
		uesRSRP := GetUEsRsrpBasedOnLocation(uesLocations, updatedCells, ueHeight)
		ueList = make(map[string]model_ransim.UE)

		for sCellNCGI, cqiMap := range cellCqiUesMap {
			sCell, ok := updatedCells[strconv.FormatUint(sCellNCGI, 10)]
			if !ok {
				continue
			}

			totalUEs := 0
			for _, numUEs := range cqiMap {
				totalUEs += numUEs
			}

			InitBWPs(sCell, cellPrbsMap, sCellNCGI, totalUEs)

			ueBWPIndexes := PartitionIndexes(len(sCell.Bwps), totalUEs, Lognormally)

			for cqi, numUEs := range cqiMap {
				for i := 0; i < numUEs; i++ {
					if len(uesLocations[sCellNCGI][cqi]) <= i {
						log.Error("number of ue locations generated is smaller than the required")
						break
					}
					ueSINR := uesSINR[sCellNCGI][cqi]
					ueRSRP := uesRSRP[sCellNCGI][cqi][i]
					ueLocation := uesLocations[sCellNCGI][cqi][i]
					ueNeighbors := signal_ransim.GetUeNeighbors(ueLocation, sCell, updatedCells, ueHeight)
					ueRSRQ := signal_ransim.RSRQ(ueSINR, cellPrbsMap[sCellNCGI][TOTAL_PRBS_DL_METRIC])

					simUE, ueIMSI := signal_ransim.CreateSimulationUE(sCellNCGI, len(ueList)+1, cqi, ueSINR, ueRSRP, ueRSRQ, ueLocation, ueNeighbors)
					simUE.Cell.BwpRefs = GetBWPRefs(ueBWPIndexes[i])
					ueList[ueIMSI] = *simUE

				}
			}
		}
	} else {
		ueList = ueGroup
	}

	log.Infof("------------- len(ueList): %d --------------", len(ueList))
	log.Infof("---------------- Updated UEs -----------------")
	return ueList, storeInCache
}

func CreateCellInfoMaps(cellMeasurements []*metrics_ransim.Metric) (map[uint64]map[int]int, map[uint64]map[string]int) {
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
		if matchesPattern(metric.Key, USED_PRBS_DL_METRIC) {
			usedPrbsDl, _ := strconv.Atoi(metric.GetValue())
			m := strings.Join(strings.Split(USED_PRBS_DL_METRIC, ".")[0:1], ".")
			cellPrbsMap[metric.EntityID][m] += usedPrbsDl
		}
		if matchesPattern(metric.Key, USED_PRBS_UL_METRIC) {
			usedPrbsUl, _ := strconv.Atoi(metric.GetValue())
			m := strings.Join(strings.Split(USED_PRBS_UL_METRIC, ".")[0:1], ".")
			cellPrbsMap[metric.EntityID][m] += usedPrbsUl
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

func GenerateUEsLocationBasedOnCQI(cellCqiUesMap map[uint64]map[int]int, simModelCells map[string]*model_ransim.Cell, ueHeight, dc float64) (map[uint64]map[int][]model_ransim.Coordinate, map[uint64]map[int]float64) {
	// map[servingCellNCGI]map[CQI][]Locations
	uesLocations := make(map[uint64]map[int][]model_ransim.Coordinate)

	// map[servingCellNCGI]map[CQI]SINR
	uesSINR := make(map[uint64]map[int]float64)

	var wg sync.WaitGroup

	for sCellNCGI, cqiMap := range cellCqiUesMap {

		if _, exists := uesSINR[sCellNCGI]; !exists {
			uesSINR[sCellNCGI] = make(map[int]float64)
		}
		if _, exists := uesLocations[sCellNCGI]; !exists {
			uesLocations[sCellNCGI] = make(map[int][]model_ransim.Coordinate)
		}

		for cqi, numUEs := range cqiMap {
			wg.Add(1)
			go func(sCellNCGI uint64, cqi, numUEs int) {
				defer wg.Done()
				ueSINR := signal_ransim.GetSINR(cqi)

				ueLocationForCqi := signal_ransim.GenerateUEsLocations(sCellNCGI, numUEs, cqi, ueSINR, ueHeight, dc, simModelCells)
				uesLocations[sCellNCGI][cqi] = ueLocationForCqi
				uesSINR[sCellNCGI][cqi] = ueSINR
			}(sCellNCGI, cqi, numUEs)
		}

	}
	wg.Wait()

	return uesLocations, uesSINR
}

func GetUEsRsrpBasedOnLocation(uesLocations map[uint64]map[int][]model_ransim.Coordinate, simModelCells map[string]*model_ransim.Cell, ueHeight float64) map[uint64]map[int][]float64 {

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

		K := 0.0
		if sCell.Channel.LOS {
			K = rand.NormFloat64()*signal_ransim.RICEAN_K_STD_MACRO + signal_ransim.RICEAN_K_MEAN
		}
		mpf := signal_ransim.RiceanFading(K)

		for cqi, cellUesLocations := range cqiMap {
			uesRSRP[sCellNCGI][cqi] = make([]float64, len(uesLocations[sCellNCGI][cqi]))
			for i, ueCoord := range cellUesLocations {
				uesRSRP[sCellNCGI][cqi][i] = signal_ransim.Strength(ueCoord, ueHeight, mpf, *sCell)
			}

		}

	}
	return uesRSRP
}
