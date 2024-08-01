package signal

import (
	"context"

	redisLib "github.com/nfvri/ran-simulator/pkg/store/redis"

	"github.com/nfvri/ran-simulator/pkg/model"
	log "github.com/sirupsen/logrus"
)

func ΙnitCoverageAndShadowMaps(cellList []*model.Cell, redisStore redisLib.RedisStore, ueHeight, refSignalStrength, dc float64) {

	ctx := context.Background()

	for i, cell := range cellList {

		cachedCell, err := redisStore.Get(ctx, cell.NCGI)

		if err != nil || !cell.ConfigEquivalent(cachedCell) {
			updated := updateCellParams(ueHeight, cell, refSignalStrength, dc)
			if !updated {
				continue
			}

			redisStore.Add(ctx, cell)
		} else {
			cellList[i] = cachedCell
		}
	}

	for i := 0; i < len(cellList); i++ {
		for j := i + 1; j < len(cellList); j++ {
			replaceOverlappingShadowMapValues(cellList[i], cellList[j])
		}
	}
	log.Infof("Updated Cells from store")

}

func updateCellParams(ueHeight float64, cell *model.Cell, refSignalStrength, d_c float64) bool {
	rpBoundaryPoints := GetRPBoundaryPoints(ueHeight, cell, refSignalStrength)
	if len(rpBoundaryPoints) == 0 {
		return false
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
		return false
	}
	cell.CoverageBoundaries = []model.CoverageBoundary{
		{
			RefSignalStrength: refSignalStrength,
			BoundaryPoints:    covBoundaryPoints,
		},
	}

	return true
}
