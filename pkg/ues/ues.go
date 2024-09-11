package ues

import (
	"context"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	model_ransim "github.com/nfvri/ran-simulator/pkg/model"
	signal_ransim "github.com/nfvri/ran-simulator/pkg/signal"
	redis_ransim "github.com/nfvri/ran-simulator/pkg/store/redis"
	metrics_ransim "github.com/onosproject/onos-api/go/onos/ransim/metrics"
	log "github.com/sirupsen/logrus"
)

const ACTIVE_UES_METRIC = "DRB.MeanActiveUeDl."
const TOTAL_PRBS_DL_METRIC = "RRU.PrbAvailDl"
const TOTAL_PRBS_UL_METRIC = "RRU.PrbAvailUl"

func InitUEs(cellMeasurements []*metrics_ransim.Metric, updatedCells map[string]*model_ransim.Cell, cacheStore redis_ransim.Store, snapshotId string, dc, ueHeight float64) (map[string]model_ransim.UE, bool) {

	cellCqiUesMap, cellTotalPrbsDlMap := CreateCellInfoMaps(cellMeasurements)

	storeInCache := false
	var ueList map[string]model_ransim.UE

	ctx := context.Background()
	ueGroup, err := cacheStore.GetUEGroup(ctx, snapshotId)
	if err != nil {
		if snapshotId != "" {
			// Add ueGroup in redis only if a new snapshot is created
			// Don't add ueGroup in redis if GenerateUEs is called in visualize liveSnapshot
			storeInCache = true
		}
		uesLocations, uesSINR := GenerateUEsLocationBasedOnCQI(cellCqiUesMap, updatedCells, ueHeight, dc)
		uesRSRP := GetUEsRsrpBasedOnLocation(uesLocations, updatedCells, ueHeight)
		ueList = make(map[string]model_ransim.UE)

		for sCellNCGI, cqiMap := range cellCqiUesMap {
			sCell, ok := updatedCells[strconv.FormatUint(sCellNCGI, 10)]
			if !ok {
				continue
			}
			totalPrbsDl, allocation := getBwpsAllocation(sCell, cellTotalPrbsDlMap, cqiMap)

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
					ueRSRQ := signal_ransim.RSRQ(ueSINR, totalPrbsDl)

					simUE, ueIMSI := signal_ransim.CreateSimulationUE(sCellNCGI, len(ueList)+1, cqi, ueSINR, ueRSRP, ueRSRQ, ueLocation, ueNeighbors)
					simUE.Cell.BwpRefs = allocateBwps(allocation[i])
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

func getBwpsAllocation(sCell *model_ransim.Cell, cellTotalPrbsDlMap map[uint64]int, cqiMap map[int]int) (int, [][]int) {

	totalPrbsDl, ok := cellTotalPrbsDlMap[uint64(sCell.NCGI)]
	if !ok {
		totalPrbsDl = 133
	}

	totalUEs := 0
	for _, numUEs := range cqiMap {
		totalUEs += numUEs
	}
	bwps := make([]model_ransim.Bwp, len(sCell.Bwps))
	for _, value := range sCell.Bwps {
		bwps = append(bwps, value)
	}
	if len(bwps) == 0 {
		bwps = SplitInParts(sCell.Channel.BsChannelBwDL+sCell.Channel.BsChannelBwUL, totalUEs)
		sCell.Bwps = make(map[string]model_ransim.Bwp)
		for _, bwp := range bwps {
			sCell.Bwps[bwp.ID] = bwp
		}
	}

	allocation := partitionIndexes(len(bwps), totalUEs, lognormally)
	return totalPrbsDl, allocation
}

func CreateCellInfoMaps(cellMeasurements []*metrics_ransim.Metric) (map[uint64]map[int]int, map[uint64]int) {
	cellCQIUEsMap := map[uint64]map[int]int{}
	cellTotalPrbsDlMap := map[uint64]int{}
	for _, metric := range cellMeasurements {
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
			cellTotalPrbsDlMap[metric.EntityID] = totalPrbsDl
		}
	}
	return cellCQIUEsMap, cellTotalPrbsDlMap
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

func SplitInParts(totalAvailableBwMHz uint32, numberOfUEs int) []model_ransim.Bwp {

	// totalAvailableBwHz := int(totalAvailableBwMHz) * 1e6
	totalAvailableBwHz := int(50) * 1e6
	bwps := []model_ransim.Bwp{}
	remainingBW := totalAvailableBwHz

	// SCS options in kHz
	scsOptions := []int{15_000} //, 30_000, 60_000, 120_000}

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Minimum PRBs constraint
	const minPRBs = 1

	// Calculate the target bandwidth per UE
	bwPerUE := remainingBW / numberOfUEs

	log.Infof("bwPerUE: %v", bwPerUE)
	bwpCount := 0
	for i := 0; i < numberOfUEs; i++ {
		if remainingBW <= 0 {
			break
		}

		// Randomly select an SCS
		scs := scsOptions[rand.Intn(len(scsOptions))]
		maxPRBs := bwPerUE / (scs * 12)

		if maxPRBs >= minPRBs {
			// Randomly select the number of PRBs ensuring it meets the minimum constraint
			prbs := rand.Intn(maxPRBs-minPRBs+1) + minPRBs
			allocatedBW := prbs * scs * 12

			// Ensure allocatedBW does not exceed the remaining bandwidth
			if allocatedBW > remainingBW {
				allocatedBW = remainingBW
				prbs = allocatedBW / (scs * 12)
			}
			bwps = append(bwps, model_ransim.Bwp{ID: strconv.Itoa(bwpCount), NumberOfRBs: prbs, Scs: scs})
			remainingBW -= allocatedBW
		} else if len(bwps) > 0 {
			// Adjust the last allocation to match the remaining bandwidth if it's smaller than minimum PRBs
			lastIndex := len(bwps) - 1
			lastPRBs := bwps[lastIndex].NumberOfRBs
			lastSCS := bwps[lastIndex].Scs
			newPRBs := lastPRBs + (remainingBW / (lastSCS * 12))
			bwps[lastIndex] = model_ransim.Bwp{ID: strconv.Itoa(bwpCount), NumberOfRBs: newPRBs, Scs: lastSCS}
			remainingBW = 0
		} else {
			// If no allocations have been made and remainingBW is too small, break
			break
		}
		bwpCount++
	}

	// Check for total bandwidth correctness
	totalAllocatedBW := 0
	for _, alloc := range bwps {
		totalAllocatedBW += alloc.NumberOfRBs * alloc.Scs * 12
	}

	if totalAllocatedBW == totalAvailableBwHz {
		log.Info("Allocation successful, total bandwidth matched.")
	} else {
		log.Infof("Allocation mismatch: allocated %d Hz, expected %d Hz\n", totalAllocatedBW, totalAvailableBwHz)
	}
	// log.Infof("Allocated PRBs and SCS: %+v", bwps)
	return bwps
}

func allocateBwps(indexes []int) []string {
	bwpRefs := make([]string, len(indexes))
	for i, indx := range indexes {
		bwpRefs[i] = strconv.Itoa(indx)
	}
	return bwpRefs
}

func partitionIndexes(n int, k int, generateSize func(int, int) int) [][]int {
	// Create a slice of indexes from 0 to n-1
	indexes := make([]int, n)
	for i := 0; i < n; i++ {
		indexes[i] = i
	}

	// Shuffle the indexes to ensure random distribution
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(n, func(i, j int) {
		indexes[i], indexes[j] = indexes[j], indexes[i]
	})

	// Generate sizes for each part
	partSizes := make([]int, k)
	remaining := n

	for i := 0; i < k; i++ {
		if i == k-1 {
			// Assign the remaining indexes to the last part
			partSizes[i] = remaining
		} else {
			// Generate a size for the current part
			size := generateSize(remaining, k-i)
			if size > remaining {
				size = remaining
			}
			partSizes[i] = size
			remaining -= size
		}
	}

	// Partition the shuffled indexes based on the generated sizes
	parts := make([][]int, k)
	start := 0
	for i := 0; i < k; i++ {
		if start+partSizes[i] > len(indexes) {
			// Ensure we don't exceed bounds
			partSizes[i] = len(indexes) - start
		}
		parts[i] = indexes[start : start+partSizes[i]]
		start += partSizes[i]
	}

	return parts
}

func lognormally(remaining int, partsLeft int) int {
	// Generate a lognormally distributed value with mean=0 and stddev=1
	mean := 0.0
	stddev := 1.0

	logNormalValue := math.Exp(rand.NormFloat64()*stddev + mean)

	// Scale the lognormal value to fit within the remaining size
	size := int(logNormalValue / float64(partsLeft) * float64(remaining))

	// Ensure size is at least 1 and does not exceed remaining
	if size < 1 {
		size = 1
	} else if size > remaining {
		size = remaining
	}

	return size
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
