package signal

import (
	"context"
	"math"

	"github.com/nfvri/ran-simulator/pkg/model"
	redisLib "github.com/nfvri/ran-simulator/pkg/store/redis"
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
	// for _, cell := range cellList {
	// 	log.Debug("*******************")
	// 	log.Debug(cell.NCGI)
	// 	log.Debug("*******************")
	// 	gridSize := int(math.Sqrt(float64(len(cell.GridPoints)))) - 1
	// 	fmt.Printf("%5v,", "i\\j")
	// 	for i := 0; i < gridSize; i++ {
	// 		fmt.Printf("%8d,", i)
	// 	}
	// 	log.Debug()
	// 	for i := 0; i < gridSize; i++ {
	// 		fmt.Printf("%5d,", i)
	// 		for j := 0; j < gridSize; j++ {

	// 			fmt.Printf("%8.4f,", cell.ShadowingMap[i][j])
	// 		}
	// 		log.Debug()
	// 	}
	// }
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

func replaceOverlappingShadowMapValues(cell1 *model.Cell, cell2 *model.Cell) {
	cell1iList, cell1jList, cell2iList, cell2jList, overlapping := FindOverlappingGridPoints(cell1.GridPoints, cell2.GridPoints)
	if overlapping {
		if cell1.NCGI == cell2.NCGI {
			log.Debugf("%d and %d overlapping but is the same cell\n", cell1.NCGI, cell2.NCGI)
		} else {
			for i := range cell1iList {
				log.Debugf("%d and %d overlapping: (%d,%d) and (%d,%d)\n", cell1.NCGI, cell2.NCGI, cell1iList[i], cell1jList[i], cell2iList[i], cell2jList[i])
				cell2.ShadowingMap[cell2iList[i]][cell2jList[i]] = cell1.ShadowingMap[cell1iList[i]][cell1jList[i]]
			}
		}
	} else {
		log.Debugf("%d and %d does not overlap\n", cell1.NCGI, cell2.NCGI)
	}
}

func DbmToMw(dbm float64) float64 {
	return math.Pow(10, dbm/10)
}

func MwToDbm(mw float64) float64 {
	return 10 * math.Log10(mw)
}
