package signal

import (
	"context"
	"strconv"
	"sync"

	redisLib "github.com/nfvri/ran-simulator/pkg/store/redis"
	"github.com/onosproject/onos-api/go/onos/ransim/types"

	"github.com/nfvri/ran-simulator/pkg/model"
	log "github.com/sirupsen/logrus"
)

func UpdateCells(cellGroup map[string]*model.Cell, redisStore redisLib.Store, ueHeight, refSignalStrength, dc float64, snapshotId string) bool {

	ctx := context.Background()
	var wg sync.WaitGroup
	storeInCache := false
	cachedCells := map[types.NCGI]struct{}{}

	cachedCellGroup, err := redisStore.GetCellGroup(ctx, snapshotId)
	// Add cellGroup in redis only if a new snapshot is created
	// Don't add cellGroup in redis if UpdateCells is called in visualize liveSnapshot
	storeInCache = (snapshotId != "") && (err != nil)
	if err != nil {

		for _, cell := range cellGroup {
			wg.Add(1)
			go func(cell *model.Cell) {
				defer wg.Done()
				updateCellParams(ueHeight, cell, refSignalStrength, dc)
			}(cell)
		}
	} else {
		for _, cell := range cellGroup {
			ncgi := strconv.FormatUint(uint64(cell.NCGI), 10)
			cachedCell, ok := cachedCellGroup[ncgi]
			if !ok || !cell.ConfigEquivalent(&cachedCell) {
				wg.Add(1)
				go func(cell *model.Cell) {
					defer wg.Done()
					updateCellParams(ueHeight, cell, refSignalStrength, dc)
				}(cell)

			} else {
				cellGroup[ncgi] = &cachedCell
				cachedCell.Cached = true
				cachedCells[cell.NCGI] = struct{}{}
			}

		}
	}

	wg.Wait()

	cellList := []*model.Cell{}
	for _, cell := range cellGroup {
		cellList = append(cellList, cell)
	}

	for i := 0; i < len(cellList); i++ {
		_, isCachedcellI := cachedCells[cellList[i].NCGI]
		for j := len(cellList) - 1; j > i; j-- {
			_, isCachedcellJ := cachedCells[cellList[j].NCGI]
			if !isCachedcellI || !isCachedcellJ {
				replaceOverlappingShadowMapValues(cellList[i], cellList[j])
			}
		}
	}

	log.Infof("---------------- Updated Cells ---------------")
	return storeInCache
}

func updateCellParams(ueHeight float64, cell *model.Cell, refSignalStrength, dc float64) {
	rpBoundaryPoints := GetRPBoundaryPoints(ueHeight, cell, refSignalStrength)
	if len(rpBoundaryPoints) == 0 {
		log.Errorf("failed to update cell's: %v rpBoundaryPoints", cell.NCGI)
		return
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
		log.Errorf("failed to update cell's: %v covBoundaryPoints", cell.NCGI)
		return
	}
	log.Infof("NCGI: %v: len(covBoundaryPoints): %d", cell.NCGI, len(covBoundaryPoints))
	cell.CoverageBoundaries = []model.CoverageBoundary{
		{
			RefSignalStrength: refSignalStrength,
			BoundaryPoints:    covBoundaryPoints,
		},
	}
}

func InitUEs(m *model.Model, redisStore redisLib.Store) {
	ctx := context.Background()

	if m.SnapshotId == "" {
		return
	}

	ueList, err := redisStore.GetUEGroup(ctx, m.SnapshotId)
	if err != nil {
		log.Errorf("failed to get ue list from redis:%v", err)
		return
	}

	m.UEList = ueList
	log.Infof("len(m.UEList): %v", len(m.UEList))
}
