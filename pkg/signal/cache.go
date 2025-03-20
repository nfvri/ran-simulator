package signal

import (
	"context"
	"strconv"
	"sync"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
	redisLib "github.com/nfvri/ran-simulator/pkg/store/redis"

	"github.com/nfvri/ran-simulator/pkg/model"
	log "github.com/sirupsen/logrus"
)

func UpdateCells(cellGroup map[string]*model.Cell, redisStore redisLib.Store, ueHeight, refSignalStrength, dc float64, snapshotId string) bool {

	ctx := context.Background()
	var wg sync.WaitGroup
	storeInCache := false
	shouldUpdateCellGroup := false
	cachedCells := map[types.NCGI]struct{}{}

	updateCell := func(snapShotCell, cachedCell *model.Cell) {
		shouldUpdateCellGroup = true
		wg.Add(1)
		go func(snapshotCell, cachedcell *model.Cell) {
			defer wg.Done()
			updateCellParams(snapshotCell, cachedcell, ueHeight, refSignalStrength, dc)
		}(snapShotCell, cachedCell)
	}

	cachedCellGroup, err := redisStore.GetCellGroup(ctx, snapshotId)
	cellGroupIncache := err == nil

	for _, cell := range cellGroup {
		for i, carrier := range cell.GetCellConfig().Carriers {
			log.Infof("%v --> before cache cell.cellConfig.Carrier[%v].TxPowerDB: %v", cell.NCGI, i, carrier.TxPowerDB)
		}
		if !cellGroupIncache {
			updateCell(cell, nil)
			continue
		}

		ncgi := strconv.FormatUint(uint64(cell.NCGI), 10)
		cachedCell, cellInCache := cachedCellGroup[ncgi]
		if !cellInCache {
			updateCell(cell, nil)
			continue
		}

		_, curCellConfigInCache := cachedCell.CachedStates[cell.GetHashedConfig()]
		if !curCellConfigInCache {
			updateCell(cell, &cachedCell)
			continue
		}

		cell.CachedStates = cachedCell.CachedStates
		cell.Bwps = cachedCell.Bwps
		cell.InterferingBeams = cachedCell.InterferingBeams
		cell.Grid = cachedCell.Grid
		cell.CurrentStateHash = cell.GetHashedConfig()
		cachedCells[cell.NCGI] = struct{}{}

	}

	wg.Wait()
	// Add cellGroup in redis only if a new snapshot is created
	// Don't add cellGroup in redis if UpdateCells is called in visualize liveSnapshot
	storeInCache = (snapshotId != "") && shouldUpdateCellGroup

	cellList := []*model.Cell{}
	for _, cell := range cellGroup {
		cellList = append(cellList, cell)
	}

	for i := 0; i < len(cellList); i++ {
		_, isCachedCellI := cachedCells[cellList[i].NCGI]
		for j := i + 1; j < len(cellList); j++ {
			_, isCachedCellJ := cachedCells[cellList[j].NCGI]

			if isCachedCellI && isCachedCellJ {
				continue
			}

			for carrierIndexI, carrierI := range cellList[i].Carriers {
				carIndexI := carrierIndexI + 1
				for beamIndexI := range carrierI.Beams {
					bmIndexI := beamIndexI + 1
					beamIDI := model.BeamID{NCGI: cellList[i].NCGI, CarrierIndex: carIndexI, BeamIndex: bmIndexI}

					for carrierIndexJ, carrierJ := range cellList[j].Carriers {
						carIndexJ := carrierIndexJ + 1
						for beamIndexJ := range carrierJ.Beams {
							bmIndexJ := beamIndexJ + 1
							beamIDJ := model.BeamID{NCGI: cellList[j].NCGI, CarrierIndex: carIndexJ, BeamIndex: bmIndexJ}
							replaceOverlappingShadowMapValues(cellList[i], cellList[j], beamIDI, beamIDJ)
						}
					}

				}
			}

		}
	}

	log.Infof("---------------- Updated Cells ---------------")
	return storeInCache
}

func updateCellParams(snapShotCell, cachedCell *model.Cell, ueHeight, refSignalStrength, dc float64) {

	if cachedCell != nil {
		snapShotCell.CachedStates = cachedCell.CachedStates
		snapShotCell.Bwps = cachedCell.Bwps
		snapShotCell.InterferingBeams = cachedCell.InterferingBeams
		snapShotCell.Grid = cachedCell.Grid
	} else {
		snapShotCell.CachedStates = make(map[string]*model.CellCoverageInfo)
		snapShotCell.Grid.BoundingBoxes = make(map[model.BeamID]*model.BoundingBox)
		snapShotCell.Grid.GridPoints = make(map[model.BeamID][]model.Coordinate)
		snapShotCell.Grid.ShadowingMaps = make(map[model.BeamID][]float64)
		snapShotCell.InterferingBeams = make(map[model.BeamID][]model.BeamID)
	}

	snapShotCell.CurrentStateHash = snapShotCell.GetHashedConfig()
	snapShotCell.CachedStates[snapShotCell.CurrentStateHash] = &model.CellCoverageInfo{
		RPCoverageBoundaries: map[model.BeamID][]model.CoverageBoundary{},
		CoverageBoundaries:   map[model.BeamID][]model.CoverageBoundary{},
	}

	for carrierIndex, carrier := range snapShotCell.Carriers {
		carIndex := carrierIndex + 1
		for beamIndex := range carrier.Beams {
			bmIndex := beamIndex + 1
			beamID := model.BeamID{NCGI: snapShotCell.NCGI, CarrierIndex: carIndex, BeamIndex: bmIndex}
			rpBoundaryPoints := GetRPBoundaryPoints(snapShotCell, beamID, refSignalStrength, ueHeight)
			if len(rpBoundaryPoints) == 0 && carrier.TxPowerDB != 0 {
				log.Errorf("failed to update cell's '%v' carrier's '%v'beam's '%v' beamrpBoundaryPoints", snapShotCell.NCGI, carIndex, bmIndex)
				return
			}
			rpBoundaryPointsFiltered := FilterBoundaryPoints(rpBoundaryPoints, carrier.Center)
			snapShotCell.CachedStates[snapShotCell.CurrentStateHash].RPCoverageBoundaries[beamID] = []model.CoverageBoundary{
				{
					RefSignalStrength: refSignalStrength,
					BoundaryPoints:    rpBoundaryPointsFiltered,
				},
			}

			InitShadowMap(snapShotCell, beamID, dc)
			carrier := snapShotCell.GetCarrier(beamID)
			covBoundaryPoints := GetCovBoundaryPoints(snapShotCell, beamID, ueHeight, refSignalStrength, rpBoundaryPoints)
			if len(covBoundaryPoints) == 0 && carrier.TxPowerDB != 0 {
				log.Errorf("failed to update cell's: %v covBoundaryPoints", snapShotCell.NCGI)
				return
			}
			covBoundaryPoints = FilterBoundaryPoints(covBoundaryPoints, carrier.Center)
			log.Infof("NCGI: %v: len(covBoundaryPoints): %d", snapShotCell.NCGI, len(covBoundaryPoints))
			snapShotCell.CachedStates[snapShotCell.CurrentStateHash].CoverageBoundaries[beamID] = []model.CoverageBoundary{
				{
					RefSignalStrength: refSignalStrength,
					BoundaryPoints:    covBoundaryPoints,
				},
			}
		}
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

	m.UEs = make(map[string]*model.UE)
	for imsi := range ueList {
		ue := ueList[imsi]
		m.UpsertUE(&ue)
	}
	log.Infof("len(m.UEList): %v", len(m.UEs))
}
