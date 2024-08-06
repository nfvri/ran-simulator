package signal

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	redisLib "github.com/nfvri/ran-simulator/pkg/store/redis"

	"github.com/nfvri/ran-simulator/pkg/model"
	log "github.com/sirupsen/logrus"
)

func UpdateCells(cellList []*model.Cell, redisStore *redisLib.RedisStore, ueHeight, refSignalStrength, dc float64, snapshotId string) {

	ctx := context.Background()
	var wg sync.WaitGroup
	var mu sync.Mutex
	newSnapshot := false

	cachedCellGroup, err := redisStore.GetCellGroup(ctx, snapshotId)
	if err != nil {
		if snapshotId != "" {
			// Add cellGroup in redis only if a new snapshot is created
			// Don't add cellGroup in redis if InitCoverageAndShadowMaps is called in visualize liveSnapshot
			newSnapshot = true
		}
		for _, cell := range cellList {
			wg.Add(1)
			go func(cell *model.Cell) {
				defer wg.Done()
				if err := updateCellParams(ueHeight, cell, refSignalStrength, dc); err != nil {
					return
				}

			}(cell)
		}
	} else {

		for i, cell := range cellList {
			ncgi := strconv.FormatUint(uint64(cell.NCGI), 10)
			cachedCell, ok := cachedCellGroup[ncgi]
			if !ok || !cell.ConfigEquivalent(&cachedCell) {
				wg.Add(1)
				go func(cell *model.Cell) {
					defer wg.Done()
					if err := updateCellParams(ueHeight, cell, refSignalStrength, dc); err != nil {
						return
					}
				}(cell)

			} else {
				mu.Lock()
				cellList[i] = &cachedCell
				mu.Unlock()
			}

		}
	}

	wg.Wait()

	for i := 0; i < len(cellList); i++ {
		for j := len(cellList) - 1; j > i; j-- {
			replaceOverlappingShadowMapValues(cellList[i], cellList[j])
		}
	}
	if newSnapshot {
		cellGroup := make(map[string]model.Cell)
		for _, cell := range cellList {
			ncgi := strconv.FormatUint(uint64(cell.NCGI), 10)
			cellGroup[ncgi] = *cell
		}
		redisStore.AddCellGroup(ctx, snapshotId, cellGroup)
		log.Infof("---------- Added CellGroup in Cache ----------")
	}
	log.Infof("---------------- Updated Cells ---------------")

}

func updateCellParams(ueHeight float64, cell *model.Cell, refSignalStrength, dc float64) error {
	rpBoundaryPoints := GetRPBoundaryPoints(ueHeight, cell, refSignalStrength)
	if len(rpBoundaryPoints) == 0 {
		return fmt.Errorf("failed to update cell")
	}
	cell.RPCoverageBoundaries = []model.CoverageBoundary{
		{
			RefSignalStrength: refSignalStrength,
			BoundaryPoints:    rpBoundaryPoints,
		},
	}
	InitShadowMap(cell, dc)
	covBoundaryPoints := GetCovBoundaryPoints(ueHeight, cell, refSignalStrength, rpBoundaryPoints)

	if len(covBoundaryPoints) == 0 {
		return fmt.Errorf("failed to update cell")
	}
	cell.CoverageBoundaries = []model.CoverageBoundary{
		{
			RefSignalStrength: refSignalStrength,
			BoundaryPoints:    covBoundaryPoints,
		},
	}

	return nil
}
