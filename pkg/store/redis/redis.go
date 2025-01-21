package cells

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

type Store interface {
	AddCellGroup(ctx context.Context, snapshotId string, cellGroupPtr map[string]*model.Cell) error
	GetCellGroup(ctx context.Context, snapshotId string) (map[string]model.Cell, error)
	DeleteCellGroup(ctx context.Context, snapshotId string) (map[string]model.Cell, error)
	AddUEGroup(ctx context.Context, snapshotId string, ueGroup map[string]*model.UE) error
	GetUEGroup(ctx context.Context, snapshotId string) (map[string]model.UE, error)
	DeleteUEGroup(ctx context.Context, snapshotId string) (map[string]model.UE, error)
}

type MockedRedisStore struct {
	cache map[string]map[string]model.Cell
}

func (s *MockedRedisStore) AddCellGroup(ctx context.Context, snapshotId string, cellGroupPtr map[string]*model.Cell) error {
	if len(s.cache) == 0 {
		s.cache = make(map[string]map[string]model.Cell)
	}
	if snapshotId != "" {
		cellGroup := make(map[string]model.Cell)
		for ncgi, cell := range cellGroupPtr {
			cellGroup[ncgi] = *cell
		}
		s.cache[snapshotId] = cellGroup
	}
	return nil
}
func (s *MockedRedisStore) GetCellGroup(ctx context.Context, snapshotId string) (map[string]model.Cell, error) {
	if snapshotId == "test" {
		path, _ := os.Getwd()
		idx := strings.Index(path, "ran-simulator")
		path = path[:idx]

		byteValue, err := os.ReadFile(path + "ran-simulator/pkg/testdata/cells_test.json")
		if err != nil {
			log.Fatalf("Failed to open JSON file: %s", err)
		}

		var cellGroup map[string]model.Cell
		if err := json.Unmarshal(byteValue, &cellGroup); err != nil {
			log.Fatalf("Failed to unmarshal JSON: %s", err)
		}
		return cellGroup, nil
	} else if snapshotId != "" {
		cellGroup, ok := s.cache[snapshotId]

		if ok {
			// cellGroupPtr := make(map[string]*model.Cell)
			// for ncgi, cell := range cellGroup {
			// 	cellGroupPtr[ncgi] = &cell
			// }
			return cellGroup, nil
		}

	}
	return nil, fmt.Errorf("no entry exists in cache")

}
func (s *MockedRedisStore) DeleteCellGroup(ctx context.Context, snapshotId string) (map[string]model.Cell, error) {
	return make(map[string]model.Cell), nil
}
func (s *MockedRedisStore) AddUEGroup(ctx context.Context, snapshotId string, ueGroup map[string]*model.UE) error {
	return nil
}
func (s *MockedRedisStore) GetUEGroup(ctx context.Context, snapshotId string) (map[string]model.UE, error) {
	return nil, fmt.Errorf("no entry exists in cache")
}
func (s *MockedRedisStore) DeleteUEGroup(ctx context.Context, snapshotId string) (map[string]model.UE, error) {
	return make(map[string]model.UE), nil
}

type RedisStore struct {
	CellDB *redis.Client
	UeDB   *redis.Client
}

func InitClient(redisHost, redisPort, db, username, password string) *redis.Client {

	database, err := strconv.Atoi(db)
	if err != nil {
		log.Error(err)
		return nil
	}
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Username: username,
		Password: password,
		DB:       database,
	})
}

type RedisGrid struct {
	ShadowingMaps map[string][]float64          `json:"shadowingMap"`
	GridPoints    map[string][]model.Coordinate `json:"gridPoints"`
	BoundingBoxes map[string]*model.BoundingBox `json:"boundingBox"`
}

type RedisCellCoverageInfo struct {
	RPCoverageBoundaries map[string][]model.CoverageBoundary `mapstructure:"rpCoverageBoundaries"`
	CoverageBoundaries   map[string][]model.CoverageBoundary `mapstructure:"coverageBoundaries"`
}

type RedisCell struct {
	model.CellConfig
	NCGI                types.NCGI              `mapstructure:"ncgi"`
	Color               string                  `mapstructure:"color"`
	MaxUEs              uint32                  `mapstructure:"maxUEs"`
	Neighbors           []types.NCGI            `mapstructure:"neighbors"`
	MeasurementParams   model.MeasurementParams `mapstructure:"measurementParams"`
	PCI                 uint32                  `mapstructure:"pci"`
	Earfcn              uint32                  `mapstructure:"earfcn"`
	CellType            types.CellType          `mapstructure:"cellType"`
	ArfcnDL             uint32                  `mapstructure:"arfcndl"`
	ArfcnUL             uint32                  `mapstructure:"arfcnul"`
	BsChannelBwDL       uint32                  `json:"bSChannelBwDL"`
	BsChannelBwUL       uint32                  `json:"bSChannelBwUL"`
	Bwps                map[uint64]*model.Bwp   `mapstructure:"bwps"`
	RrcIdleCount        uint32
	RrcConnectedCount   uint32
	Cached              bool
	CachedStates        map[string]*RedisCellCoverageInfo
	CurrentStateHash    string
	ResourceAllocScheme string
	InterferingBeams    map[string][]model.BeamID `mapstructure:"interfearingBeamsrefs"`
	RedisGrid
}

func (s *RedisStore) AddCellGroup(ctx context.Context, snapshotId string, cellGroup map[string]*model.Cell) error {

	redisCellGroup := make(map[string]RedisCell)
	for ncgi, cell := range cellGroup {
		redisCellGroup[ncgi] = RedisCell{
			CellConfig:          cell.CellConfig,
			NCGI:                cell.NCGI,
			Color:               cell.Color,
			MaxUEs:              cell.MaxUEs,
			Neighbors:           cell.Neighbors,
			MeasurementParams:   cell.MeasurementParams,
			PCI:                 cell.PCI,
			Earfcn:              cell.Earfcn,
			CellType:            cell.CellType,
			ArfcnDL:             cell.ArfcnDL,
			ArfcnUL:             cell.ArfcnUL,
			BsChannelBwDL:       cell.BsChannelBwDL,
			BsChannelBwUL:       cell.BsChannelBwUL,
			Bwps:                cell.Bwps,
			RrcIdleCount:        cell.RrcIdleCount,
			RrcConnectedCount:   cell.RrcConnectedCount,
			Cached:              cell.Cached,
			CurrentStateHash:    cell.CurrentStateHash,
			ResourceAllocScheme: cell.ResourceAllocScheme,
			CachedStates:        cellToRedisCachedStates(cell.CachedStates),
			InterferingBeams:    cellToRedisInterferingBeams(cell.InterferingBeams),
			RedisGrid:           cellToRedisGrid(cell.Grid),
		}
	}

	redisCellGroupBytes, err := json.Marshal(redisCellGroup)
	if err != nil {
		return fmt.Errorf("failed to marshal cell group: %v ", err)
	}

	return s.CellDB.Set(context.Background(), snapshotId+"-CellGroup", redisCellGroupBytes, time.Duration(0)).Err()
}

func (s *RedisStore) GetCellGroup(ctx context.Context, snapshotId string) (map[string]model.Cell, error) {
	redisCellGroupBytes, err := s.CellDB.Get(context.Background(), snapshotId+"-CellGroup").Result()
	if err != nil {
		return nil, fmt.Errorf("error fetching cell group data for snapshot id %s: %v", snapshotId, err)
	}

	if len(redisCellGroupBytes) == 0 {
		return nil, fmt.Errorf("cell group data for snapshot id %s does not exist", snapshotId)
	}

	redisCellGroup := map[string]RedisCell{}

	err = json.Unmarshal([]byte(redisCellGroupBytes), &redisCellGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cell group: %v ", err)
	}
	cellGroup := map[string]model.Cell{}

	for ncgi, redisCell := range redisCellGroup {
		cellGroup[ncgi] = model.Cell{
			CellConfig:          redisCell.CellConfig,
			NCGI:                redisCell.NCGI,
			Color:               redisCell.Color,
			MaxUEs:              redisCell.MaxUEs,
			Neighbors:           redisCell.Neighbors,
			MeasurementParams:   redisCell.MeasurementParams,
			PCI:                 redisCell.PCI,
			Earfcn:              redisCell.Earfcn,
			CellType:            redisCell.CellType,
			ArfcnDL:             redisCell.ArfcnDL,
			ArfcnUL:             redisCell.ArfcnUL,
			BsChannelBwDL:       redisCell.BsChannelBwDL,
			BsChannelBwUL:       redisCell.BsChannelBwUL,
			Bwps:                redisCell.Bwps,
			RrcIdleCount:        redisCell.RrcIdleCount,
			RrcConnectedCount:   redisCell.RrcConnectedCount,
			Cached:              redisCell.Cached,
			CurrentStateHash:    redisCell.CurrentStateHash,
			ResourceAllocScheme: redisCell.ResourceAllocScheme,
			CachedStates:        redisToCellCachedStates(redisCell.CachedStates),
			InterferingBeams:    redisToCellInterferingBeams(redisCell.InterferingBeams),
			Grid:                redisToCellGrid(redisCell.RedisGrid),
		}
	}

	return cellGroup, nil
}

func (s *RedisStore) DeleteCellGroup(ctx context.Context, snapshotId string) (map[string]model.Cell, error) {
	cellGroup, err := s.GetCellGroup(ctx, snapshotId)
	if err != nil {
		return nil, err
	}

	err = s.CellDB.Del(ctx, snapshotId+"-CellGroup").Err()
	return cellGroup, err
}

func (s *RedisStore) AddUEGroup(ctx context.Context, snapshotId string, ueGroup map[string]*model.UE) error {

	ueGroupBytes, err := json.Marshal(ueGroup)
	if err != nil {
		return fmt.Errorf("failed to marshal ue group: %v ", err)
	}

	return s.UeDB.Set(context.Background(), snapshotId+"-UEGroup", ueGroupBytes, time.Duration(0)).Err()
}

func (s *RedisStore) GetUEGroup(ctx context.Context, snapshotId string) (map[string]model.UE, error) {
	ueGroupBytes, err := s.UeDB.Get(context.Background(), snapshotId+"-UEGroup").Result()
	if err != nil {
		return nil, fmt.Errorf("error fetching ue group data for snapshot id %s: %v", snapshotId, err)
	}

	if len(ueGroupBytes) == 0 {
		return nil, fmt.Errorf("ue group data for snapshot id %s does not exist", snapshotId)
	}

	ueGroup := map[string]model.UE{}

	err = json.Unmarshal([]byte(ueGroupBytes), &ueGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal ue group: %v ", err)
	}

	return ueGroup, nil
}

func (s *RedisStore) DeleteUEGroup(ctx context.Context, snapshotId string) (map[string]model.UE, error) {
	ueGroup, err := s.GetUEGroup(ctx, snapshotId)
	if err != nil {
		return nil, err
	}

	err = s.UeDB.Del(ctx, snapshotId+"-UEGroup").Err()
	return ueGroup, err
}

func cellToRedisCachedStates(cachedStates map[string]*model.CellCoverageInfo) map[string]*RedisCellCoverageInfo {
	redisCachedStates := make(map[string]*RedisCellCoverageInfo, len(cachedStates))

	for hashedKey, cellCoverageInfo := range cachedStates {
		if cellCoverageInfo == nil {
			continue
		}
		redisCachedStates[hashedKey] = &RedisCellCoverageInfo{
			RPCoverageBoundaries: make(map[string][]model.CoverageBoundary, len(cellCoverageInfo.CoverageBoundaries)),
			CoverageBoundaries:   make(map[string][]model.CoverageBoundary, len(cellCoverageInfo.RPCoverageBoundaries)),
		}
		for beamID, RPCoverageBoundary := range cellCoverageInfo.RPCoverageBoundaries {
			beamIdStr := beamID.ToString()
			redisCachedStates[hashedKey].RPCoverageBoundaries[beamIdStr] = RPCoverageBoundary
			if covBoundary, ok := cellCoverageInfo.CoverageBoundaries[beamID]; ok {
				redisCachedStates[hashedKey].CoverageBoundaries[beamIdStr] = covBoundary
			}
		}
	}
	return redisCachedStates
}

func cellToRedisInterferingBeams(cellInterferingBeams map[model.BeamID][]model.BeamID) map[string][]model.BeamID {
	redisInterferingBeams := make(map[string][]model.BeamID, len(cellInterferingBeams))

	for beamID, interferingBeams := range cellInterferingBeams {
		beamIdStr := beamID.ToString()
		redisInterferingBeams[beamIdStr] = interferingBeams
	}
	return redisInterferingBeams
}

func cellToRedisGrid(cellGrid model.Grid) RedisGrid {
	redisGrid := RedisGrid{
		BoundingBoxes: make(map[string]*model.BoundingBox, len(cellGrid.BoundingBoxes)),
		GridPoints:    make(map[string][]model.Coordinate, len(cellGrid.GridPoints)),
		ShadowingMaps: make(map[string][]float64, len(cellGrid.ShadowingMaps)),
	}

	for beamID, boundingBox := range cellGrid.BoundingBoxes {
		beamIdStr := beamID.ToString()
		redisGrid.BoundingBoxes[beamIdStr] = boundingBox
	}

	for beamID, gridPoints := range cellGrid.GridPoints {
		beamIdStr := beamID.ToString()
		redisGrid.GridPoints[beamIdStr] = gridPoints
	}

	for beamID, shadowingMap := range cellGrid.ShadowingMaps {
		beamIdStr := beamID.ToString()
		redisGrid.ShadowingMaps[beamIdStr] = shadowingMap
	}

	return redisGrid
}

func redisToCellCachedStates(redisCachedStates map[string]*RedisCellCoverageInfo) map[string]*model.CellCoverageInfo {
	cellCachedStates := make(map[string]*model.CellCoverageInfo, len(redisCachedStates))

	for hashedKey, redisCellCoverageInfo := range redisCachedStates {
		cellCachedStates[hashedKey] = &model.CellCoverageInfo{
			RPCoverageBoundaries: make(map[model.BeamID][]model.CoverageBoundary, len(redisCellCoverageInfo.RPCoverageBoundaries)),
			CoverageBoundaries:   make(map[model.BeamID][]model.CoverageBoundary, len(redisCellCoverageInfo.CoverageBoundaries)),
		}
		for beamIdStr, RPCoverageBoundary := range redisCellCoverageInfo.RPCoverageBoundaries {
			beamID, err := model.ParseBeamID(beamIdStr)
			if err != nil {
				log.Warnf("error in parsing beam id: %v", beamIdStr)
				continue
			}
			cellCachedStates[hashedKey].RPCoverageBoundaries[beamID] = RPCoverageBoundary
			if covBoundary, ok := redisCellCoverageInfo.CoverageBoundaries[beamIdStr]; ok {
				cellCachedStates[hashedKey].CoverageBoundaries[beamID] = covBoundary
			}
		}
	}

	return cellCachedStates
}

func redisToCellInterferingBeams(redisInterferingBeams map[string][]model.BeamID) map[model.BeamID][]model.BeamID {
	cellInterferingBeams := make(map[model.BeamID][]model.BeamID, len(redisInterferingBeams))

	for beamIdStr, interferingBeams := range redisInterferingBeams {
		beamID, err := model.ParseBeamID(beamIdStr)
		if err != nil {
			continue
		}
		cellInterferingBeams[beamID] = interferingBeams
	}
	return cellInterferingBeams
}

func redisToCellGrid(redisGrid RedisGrid) model.Grid {
	cellGrid := model.Grid{
		BoundingBoxes: make(map[model.BeamID]*model.BoundingBox, len(redisGrid.BoundingBoxes)),
		GridPoints:    make(map[model.BeamID][]model.Coordinate, len(redisGrid.GridPoints)),
		ShadowingMaps: make(map[model.BeamID][]float64, len(redisGrid.ShadowingMaps)),
	}

	for beamIdStr, boundingBox := range redisGrid.BoundingBoxes {
		beamID, err := model.ParseBeamID(beamIdStr)
		if err != nil {
			continue
		}
		cellGrid.BoundingBoxes[beamID] = boundingBox
	}

	for beamIdStr, gridPoints := range redisGrid.GridPoints {
		beamID, err := model.ParseBeamID(beamIdStr)
		if err != nil {
			continue
		}
		cellGrid.GridPoints[beamID] = gridPoints
	}

	for beamIdStr, shadowingMap := range redisGrid.ShadowingMaps {
		beamID, err := model.ParseBeamID(beamIdStr)
		if err != nil {
			continue
		}
		cellGrid.ShadowingMaps[beamID] = shadowingMap
	}

	return cellGrid
}
