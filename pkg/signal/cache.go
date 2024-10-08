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
	shouldUpdateCellStates := false

	updateCell := func(snapShotCell, cachedCell *model.Cell) {
		wg.Add(1)
		go func(snapShotCell, cachedCell *model.Cell) {
			defer wg.Done()
			updateCellParams(snapShotCell, cachedCell, ueHeight, refSignalStrength, dc)
		}(snapShotCell, cachedCell)
	}

	cachedCells := map[types.NCGI]struct{}{}
	cachedCellGroup, err := redisStore.GetCellGroup(ctx, snapshotId)

	if err != nil {
		for _, cell := range cellGroup {
			updateCell(cell, nil)
		}
	} else {
		for _, cell := range cellGroup {
			ncgi := strconv.FormatUint(uint64(cell.NCGI), 10)
			cachedCell, ok := cachedCellGroup[ncgi]
			if !ok {
				shouldUpdateCellStates = true
				updateCell(cell, &cachedCell)
			}
			cellConfig := cell.GetCellConfig()
			log.Infof("%v --> cell.cellConfig.TxPowerDB: %v", cell.NCGI, cellConfig.TxPowerDB)
			_, ok = cachedCell.CachedStates[cell.GetHashedConfig()]
			if !ok {
				shouldUpdateCellStates = true
				updateCell(cell, &cachedCell)
			} else {
				cachedCell.CurrentStateHash = cell.GetHashedConfig()
				cellGroup[ncgi] = &cachedCell
				cachedCell.Cached = true
				cachedCells[cell.NCGI] = struct{}{}
			}
		}
	}

	wg.Wait()
	// Add cellGroup in redis only if a new snapshot is created
	// Don't add cellGroup in redis if UpdateCells is called in visualize liveSnapshot
	storeInCache = (snapshotId != "") && ((err != nil) || shouldUpdateCellStates)

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

func updateCellParams(snapShotCell, cachedCell *model.Cell, ueHeight, refSignalStrength, dc float64) {
	rpBoundaryPoints := GetRPBoundaryPoints(ueHeight, snapShotCell, refSignalStrength)
	if len(rpBoundaryPoints) == 0 {
		log.Errorf("failed to update cell's: %v rpBoundaryPoints", snapShotCell.NCGI)
		return
	}

	if cachedCell != nil {
		snapShotCell.CachedStates = cachedCell.CachedStates
		snapShotCell.Bwps = cachedCell.Bwps
		snapShotCell.Grid = cachedCell.Grid
	} else {
		snapShotCell.CachedStates = make(map[string]*model.CellSignalInfo)
	}

	snapShotCell.CurrentStateHash = snapShotCell.GetHashedConfig()
	snapShotCell.CachedStates[snapShotCell.CurrentStateHash] = &model.CellSignalInfo{
		RPCoverageBoundaries: []model.CoverageBoundary{
			{
				RefSignalStrength: refSignalStrength,
				BoundaryPoints:    rpBoundaryPoints,
			},
		},
	}

	InitShadowMap(snapShotCell, dc)

	covBoundaryPoints := GetCovBoundaryPoints(ueHeight, snapShotCell, refSignalStrength, rpBoundaryPoints)
	if len(covBoundaryPoints) == 0 {
		log.Errorf("failed to update cell's: %v covBoundaryPoints", snapShotCell.NCGI)
		return
	}
	log.Infof("NCGI: %v: len(covBoundaryPoints): %d", snapShotCell.NCGI, len(covBoundaryPoints))

	snapShotCell.CachedStates[snapShotCell.CurrentStateHash].CoverageBoundaries = []model.CoverageBoundary{
		{
			RefSignalStrength: refSignalStrength,
			BoundaryPoints:    covBoundaryPoints,
		},
	}
}

func PopulateUEs(m *model.Model, redisStore redisLib.Store) {
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
