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

func UpdateCells(cellGroup map[string]*model.Cell, redisStore redisLib.Store, ueHeight, refSignalStrength, dc float64, snapshotId string) bool {

	ctx := context.Background()
	var wg sync.WaitGroup
	var mu sync.Mutex
	storeInCache := false

	cachedCellGroup, err := redisStore.GetCellGroup(ctx, snapshotId)
	// Add cellGroup in redis only if a new snapshot is created
	// Don't add cellGroup in redis if UpdateCells is called in visualize liveSnapshot
	storeInCache = (snapshotId != "") && (err != nil)
	if err != nil {

		for _, cell := range cellGroup {
			wg.Add(1)
			go func(cell *model.Cell) {
				defer wg.Done()
				if err := updateCellParams(ueHeight, cell, refSignalStrength, dc); err != nil {
					return
				}

			}(cell)
		}
	} else {

		for _, cell := range cellGroup {
			ncgi := strconv.FormatUint(uint64(cell.NCGI), 10)
			cachedCell, ok := cachedCellGroup[ncgi]
			//TODO: calculate BWPs from CellMeasurements
			if !ok || !cell.ConfigEquivalent(&cachedCell) {
				cell.Bwps = cachedCell.Bwps
				wg.Add(1)
				go func(cell *model.Cell) {
					defer wg.Done()
					if err := updateCellParams(ueHeight, cell, refSignalStrength, dc); err != nil {
						return
					}
				}(cell)

			} else {
				mu.Lock()
				cellGroup[ncgi] = &cachedCell
				mu.Unlock()
			}

		}
	}

	wg.Wait()

	cellList := []*model.Cell{}
	for _, cell := range cellGroup {
		cellList = append(cellList, cell)
	}

	for i := 0; i < len(cellList); i++ {
		for j := len(cellList) - 1; j > i; j-- {
			replaceOverlappingShadowMapValues(cellList[i], cellList[j])
		}
	}

	log.Infof("---------------- Updated Cells ---------------")
	return storeInCache
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
	log.Infof("NCGI: %v: len(covBoundaryPoints): %d", cell.NCGI, len(covBoundaryPoints))
	cell.CoverageBoundaries = []model.CoverageBoundary{
		{
			RefSignalStrength: refSignalStrength,
			BoundaryPoints:    covBoundaryPoints,
		},
	}

	return nil
}
