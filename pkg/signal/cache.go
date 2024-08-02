package signal

import (
	"context"
	"fmt"
	"sync"

	redisLib "github.com/nfvri/ran-simulator/pkg/store/redis"

	"github.com/nfvri/ran-simulator/pkg/model"
	log "github.com/sirupsen/logrus"
)

func InitCoverageAndShadowMaps(cellList []*model.Cell, redisStore redisLib.RedisStore, ueHeight, refSignalStrength, dc float64) {

	ctx := context.Background()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, cell := range cellList {
		cachedCell, err := redisStore.Get(ctx, cell.NCGI)

		if err != nil || !cell.ConfigEquivalent(cachedCell) {

			wg.Add(1)
			go func(cell *model.Cell) {
				defer wg.Done()
				if err := updateCellParams(ueHeight, cell, refSignalStrength, dc); err != nil {
					return
				}

				ctx := context.Background()
				mu.Lock()
				redisStore.Add(ctx, cell)
				mu.Unlock()

			}(cell)

		} else {
			mu.Lock()
			cellList[i] = cachedCell
			mu.Unlock()
		}

	}

	wg.Wait()

	for i := 0; i < len(cellList); i++ {
		for j := len(cellList) - 1; j > i; j-- {
			replaceOverlappingShadowMapValues(cellList[i], cellList[j])
		}
	}
	log.Infof("--- Updated Cells ---")

}

func updateCellParams(ueHeight float64, cell *model.Cell, refSignalStrength, d_c float64) error {
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
	InitShadowMap(cell, d_c)
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
