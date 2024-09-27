package ues

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/nfvri/ran-simulator/pkg/model"
	log "github.com/sirupsen/logrus"
)

func InitBWPs(sCell *model.Cell, cellPrbsMap map[uint64]map[string]int, sCellNCGI uint64, totalUEs int) error {

	initialCellBwps := make(map[string]model.Bwp, len(sCell.Bwps))
	if len(sCell.Bwps) != 0 {
		for _, bwp := range sCell.Bwps {
			initialCellBwps[bwp.ID] = bwp
		}
	}

	sCell.Bwps = make(map[string]model.Bwp)
	bwps := []model.Bwp{}

	cellPrbsDl := cellPrbsMap[sCellNCGI][USED_PRBS_DL_METRIC]
	cellPrbsUl := cellPrbsMap[sCellNCGI][USED_PRBS_UL_METRIC]
	if cellPrbsDl == 0 && cellPrbsUl == 0 {
		cellPrbsDl = cellPrbsMap[sCellNCGI][TOTAL_PRBS_DL_METRIC]
		cellPrbsUl = cellPrbsMap[sCellNCGI][TOTAL_PRBS_UL_METRIC]
	}

	bwpsDl := bwpsFromPRBs(sCell, cellPrbsDl, totalUEs, true)
	bwpsUl := bwpsFromPRBs(sCell, cellPrbsUl, totalUEs, false)

	if len(bwpsDl) == 0 || len(bwpsUl) == 0 {
		if len(bwps) == 0 && sCell.Channel.BsChannelBwDL > 0 {
			bwpsDl = bwpsFromBW(sCell.Channel.BsChannelBwDL, totalUEs, true)
		}
		if len(bwps) == 0 && sCell.Channel.BsChannelBwUL > 0 {
			bwpsUl = bwpsFromBW(sCell.Channel.BsChannelBwUL, totalUEs, false)
		}
	}

	for _, bwp := range bwpsDl {
		sCell.Bwps[bwp.ID] = bwp
	}
	for _, bwp := range bwpsUl {
		bwpId, _ := strconv.Atoi(bwp.ID)
		bwp.ID = strconv.Itoa(bwpId + len(bwpsDl))
		sCell.Bwps[bwp.ID] = bwp
	}

	if len(sCell.Bwps) == 0 {
		sCell.Bwps = initialCellBwps
	}

	if len(sCell.Bwps) == 0 {
		err := fmt.Errorf("failed to initialize BWPs for simulation")
		log.Error(err)
		return err
	}

	return nil

}

func bwpsFromPRBs(sCell *model.Cell, sCellPrbs, totalUEs int, downlink bool) []model.Bwp {

	bwps := make([]model.Bwp, len(sCell.Bwps))
	scsOptions := []int{15_000, 30_000, 60_000, 120_000}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	rbsPerUe := sCellPrbs / totalUEs
	minPRBs := 1
	bwpCount := 0
	allocatedPrbs := 0

	for remainingPrbs := sCellPrbs; remainingPrbs > 0; remainingPrbs -= allocatedPrbs {
		scs := scsOptions[r.Intn(len(scsOptions))]
		maxPRBs := int(math.Max(float64(rbsPerUe/(scs/15000)), float64(1)))

		allocatedPrbs = r.Intn(maxPRBs-minPRBs+1) + minPRBs
		if allocatedPrbs <= remainingPrbs {
			bwps = append(bwps, model.Bwp{
				ID:          strconv.Itoa(bwpCount),
				NumberOfRBs: allocatedPrbs,
				Scs:         scs,
				Downlink:    downlink,
			})
			bwpCount++
		} else {
			allocatedPrbs = 0
		}
	}

	return bwps
}

func bwpsFromBW(bwMHz uint32, totalUEs int, downlink bool) []model.Bwp {

	totalAvailableBwHz := int(bwMHz) * 1e6
	bwps := []model.Bwp{}

	// SCS options in kHz
	scsOptions := []int{15_000, 30_000, 60_000, 120_000}

	// Seed the random number generator
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Minimum PRBs constraint
	const minPRBs = 1

	bwpCount := 0
	allocatedBW := 0
	// Calculate the target bandwidth per UE
	bwPerUE := totalAvailableBwHz / totalUEs

	for remainingBW := totalAvailableBwHz; remainingBW >= scsOptions[0]*12; remainingBW -= allocatedBW {

		// Randomly select an SCS
		scs := scsOptions[r.Intn(len(scsOptions))]
		maxPRBs := int(math.Max(float64(bwPerUE/(scs*12)), float64(1)))

		// Randomly select the number of PRBs ensuring it meets the minimum constraint
		allocatedPrbs := r.Intn(maxPRBs-minPRBs+1) + minPRBs
		allocatedBW = allocatedPrbs * scs * 12

		// Ensure allocatedBW does not exceed the remaining bandwidth
		if allocatedBW > remainingBW {
			allocatedBW = 0
			continue
		}

		bwps = append(bwps, model.Bwp{
			ID:          strconv.Itoa(bwpCount),
			NumberOfRBs: allocatedPrbs,
			Scs:         scs,
			Downlink:    downlink,
		})
		bwpCount++
	}

	// Check for total bandwidth correctness
	totalAllocatedBW := 0
	for _, alloc := range bwps {
		totalAllocatedBW += alloc.NumberOfRBs * alloc.Scs * 12
	}
	if totalAllocatedBW+scsOptions[0]*12 >= totalAvailableBwHz {
		log.Info("Allocation successful, total bandwidth covered.")
	} else {
		log.Infof("Allocation mismatch: allocated %d Hz, expected %d Hz\n", totalAllocatedBW, totalAvailableBwHz)
	}

	return bwps
}

func GetBWPRefs(indexes []int) []string {
	bwpRefs := make([]string, len(indexes))
	for i, indx := range indexes {
		bwpRefs[i] = strconv.Itoa(indx)
	}
	return bwpRefs
}

func PartitionIndexes(n int, k int, generateSize func(int, int) int) [][]int {
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

func Lognormally(remaining int, partsLeft int) int {
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
