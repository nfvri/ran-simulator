package bandwidth

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	log "github.com/sirupsen/logrus"
)

const TOTAL_PRBS_DL_METRIC = "RRU.PrbAvailDl"
const TOTAL_PRBS_UL_METRIC = "RRU.PrbAvailUl"
const USED_PRBS_DL_METRIC = "RRU.PrbUsedDl"
const USED_PRBS_UL_METRIC = "RRU.PrbUsedUl"

func InitBWPs(sCell *model.Cell, cellPrbsMap map[uint64]map[string]int, sCellNCGI uint64, totalUEs int) error {

	initialCellBwps := map[string]*model.Bwp{}
	if len(sCell.Bwps) != 0 {
		for index := range sCell.Bwps {
			bwp := *sCell.Bwps[index]
			initialCellBwps[bwp.ID] = &bwp
		}
	}

	cellPrbsDl := cellPrbsMap[sCellNCGI][USED_PRBS_DL_METRIC]
	cellPrbsUl := cellPrbsMap[sCellNCGI][USED_PRBS_UL_METRIC]
	if cellPrbsDl == 0 && cellPrbsUl == 0 {
		cellPrbsDl = cellPrbsMap[sCellNCGI][TOTAL_PRBS_DL_METRIC]
		cellPrbsUl = cellPrbsMap[sCellNCGI][TOTAL_PRBS_UL_METRIC]
	}

	bwpsDl := bwpsFromPRBs(cellPrbsDl, totalUEs, true)
	bwpsUl := bwpsFromPRBs(cellPrbsUl, totalUEs, false)

	if len(bwpsDl) == 0 && sCell.Channel.BsChannelBwDL > 0 {
		bwpsDl = bwpsFromBW(sCell.Channel.BsChannelBwDL, totalUEs, true)
	}
	if len(bwpsUl) == 0 && sCell.Channel.BsChannelBwUL > 0 {
		bwpsUl = bwpsFromBW(sCell.Channel.BsChannelBwUL, totalUEs, false)
	}

	sCell.Bwps = make(map[string]*model.Bwp, len(bwpsDl)+len(bwpsUl))
	for index := range bwpsDl {
		sCell.Bwps[bwpsDl[index].ID] = &bwpsDl[index]
	}

	for index := range bwpsUl {
		bwpId, _ := strconv.Atoi(bwpsUl[index].ID)
		bwpsUl[index].ID = strconv.Itoa(bwpId + len(bwpsDl))
		sCell.Bwps[bwpsUl[index].ID] = &bwpsUl[index]
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

func bwpsFromPRBs(sCellPrbs, totalUEs int, downlink bool) []model.Bwp {

	bwps := []model.Bwp{}
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

func PartitionBwps(bwps map[string]*model.Bwp, k int, generateSize func(int, int) int) [][]*model.Bwp {
	// Extract the keys from the map
	keys := make([]string, 0, len(bwps))
	for key := range bwps {
		keys = append(keys, key)
	}
	n := len(keys)

	// Shuffle the keys to ensure random distribution
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(n, func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})

	// Generate sizes for each partition
	partSizes := make([]int, k)
	remaining := n
	for i := 0; i < k; i++ {
		if i == k-1 {
			// Assign all remaining keys to the last partition
			partSizes[i] = remaining
		} else {
			// Generate size for the current partition
			size := generateSize(remaining, k-i)
			if size > remaining {
				size = remaining
			}
			partSizes[i] = size
			remaining -= size
		}
	}

	// Partition the shuffled keys based on the generated sizes
	partitions := make([][]*model.Bwp, k)
	start := 0
	for i := 0; i < k; i++ {
		end := start + partSizes[i]
		if end > n {
			end = n
		}

		// Collect Bwp pointers for this partition
		partitions[i] = []*model.Bwp{}
		for _, key := range keys[start:end] {
			partitions[i] = append(partitions[i], bwps[key])
		}
		start = end
	}

	return partitions
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

func ReleaseBWPs(sCell *model.Cell, ue *model.UE) []model.Bwp {
	bwps := make([]model.Bwp, 0, len(ue.Cell.BwpRefs))
	for index := range ue.Cell.BwpRefs {
		bwp := ue.Cell.BwpRefs[index]
		bwps = append(bwps, *bwp)
		delete(sCell.Bwps, bwp.ID)
	}
	ue.Cell.BwpRefs = []*model.Bwp{}
	return bwps
}

func AllocateBWPs(tCell *model.Cell, servedUEs []*model.UE, ue *model.UE, requestedBwps []model.Bwp) {

	if enoughBW(tCell, requestedBwps) {
		bwpId := len(tCell.Bwps)
		for index := range requestedBwps {
			bwp := requestedBwps[index]
			bwp.ID = strconv.Itoa(bwpId)
			ue.Cell.BwpRefs = append(ue.Cell.BwpRefs, &bwp)
			tCell.Bwps[bwp.ID] = &bwp
			bwpId++
		}
		return
	}

	// delete current allocation
	servedUEs = append(servedUEs, ue)
	for _, servedUe := range servedUEs {
		ReleaseBWPs(tCell, servedUe)
	}

	// reallocate using selected scheme
	switch tCell.ResourceAllocScheme {
	case PROPORTIONAL_FAIR:
		pf := ProportionalFair{
			UeBwMaxDecPerc:      0.1,
			InitialBwAllocation: tCell.InitialBwAllocation,
			CurrBwAllocation:    BwAlloctionOf(servedUEs),
		}
		pf.apply(tCell, servedUEs)
	}
}

func enoughBW(tCell *model.Cell, requestedBwps []model.Bwp) bool {
	//TODO: check if UL+DL is sufficient istead of individual checks
	requestedBWDLUe, requestedBWULUe := 0, 0
	for index := range requestedBwps {
		bwp := requestedBwps[index]
		if bwp.Downlink {
			requestedBWDLUe += bwp.Scs * 12 * bwp.NumberOfRBs
		} else {
			requestedBWULUe += bwp.Scs * 12 * bwp.NumberOfRBs
		}
	}
	usedBWDLCell, usedBWULCell := usedBWCell(tCell)

	sufficientBWDL := tCell.Channel.BsChannelBwDL-uint32(usedBWDLCell) > uint32(requestedBWDLUe)
	sufficientBWUL := tCell.Channel.BsChannelBwUL-uint32(usedBWULCell) > uint32(requestedBWULUe)

	return sufficientBWDL && sufficientBWUL
}

func usedBWCell(cell *model.Cell) (usedBWDLCell, usedBWULCell int) {

	for index := range cell.Bwps {
		bwp := cell.Bwps[index]
		if bwp.Downlink {
			usedBWDLCell += bwp.Scs * 12 * bwp.NumberOfRBs
		} else {
			usedBWULCell += bwp.Scs * 12 * bwp.NumberOfRBs
		}
	}
	return

}

func BwAlloctionOf(ues []*model.UE) map[types.IMSI][]model.Bwp {
	bwAlloc := map[types.IMSI][]model.Bwp{}
	for _, ue := range ues {
		bwAlloc[ue.IMSI] = make([]model.Bwp, 0, len(ue.Cell.BwpRefs))
		for index := range ue.Cell.BwpRefs {
			bwp := *ue.Cell.BwpRefs[index]
			bwAlloc[ue.IMSI] = append(bwAlloc[ue.IMSI], bwp)
		}
	}
	return bwAlloc
}
