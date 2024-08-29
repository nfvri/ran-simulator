package cells

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

type Store interface {
	AddCellGroup(ctx context.Context, snapshotId string, cellGroup map[string]model.Cell) error
	GetCellGroup(ctx context.Context, snapshotId string) (map[string]model.Cell, error)
	DeleteCellGroup(ctx context.Context, snapshotId string) (map[string]model.Cell, error)
	AddUEGroup(ctx context.Context, snapshotId string, ueGroup map[string]model.UE) error
	GetUEGroup(ctx context.Context, snapshotId string) (map[string]model.UE, error)
	DeleteUEGroup(ctx context.Context, snapshotId string) (map[string]model.UE, error)
}

type MockedRedisStore struct{}

func (s *MockedRedisStore) AddCellGroup(ctx context.Context, snapshotId string, cellGroup map[string]model.Cell) error {
	return nil
}
func (s *MockedRedisStore) GetCellGroup(ctx context.Context, snapshotId string) (map[string]model.Cell, error) {

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
}
func (s *MockedRedisStore) DeleteCellGroup(ctx context.Context, snapshotId string) (map[string]model.Cell, error) {
	return make(map[string]model.Cell), nil
}
func (s *MockedRedisStore) AddUEGroup(ctx context.Context, snapshotId string, ueGroup map[string]model.UE) error {
	return nil
}
func (s *MockedRedisStore) GetUEGroup(ctx context.Context, snapshotId string) (map[string]model.UE, error) {
	return make(map[string]model.UE), nil
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

func (s *RedisStore) AddCellGroup(ctx context.Context, snapshotId string, cellGroup map[string]model.Cell) error {

	cellGroupBytes, err := json.Marshal(cellGroup)
	if err != nil {
		return fmt.Errorf("failed to marshal cell group: %v ", err)
	}

	return s.CellDB.Set(context.Background(), snapshotId+"-CellGroup", cellGroupBytes, time.Duration(0)).Err()
}

func (s *RedisStore) GetCellGroup(ctx context.Context, snapshotId string) (map[string]model.Cell, error) {
	cellGroupBytes, err := s.CellDB.Get(context.Background(), snapshotId+"-CellGroup").Result()
	if err != nil {
		return nil, fmt.Errorf("error fetching cell group data for snapshot id %s: %v", snapshotId, err)
	}

	if len(cellGroupBytes) == 0 {
		return nil, fmt.Errorf("cell group data for snapshot id %s does not exist", snapshotId)
	}

	cellGroup := map[string]model.Cell{}

	err = json.Unmarshal([]byte(cellGroupBytes), &cellGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cell group: %v ", err)
	}
	for ncgi, cell := range cellGroup {
		cell.RPCoverageBoundaries = []model.CoverageBoundary{}
		cell.CoverageBoundaries = []model.CoverageBoundary{}
		cell.Grid = model.Grid{}
		cellGroup[ncgi] = cell
	}
	log.Infof("%+v", cellGroup)
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

func (s *RedisStore) AddUEGroup(ctx context.Context, snapshotId string, ueGroup map[string]model.UE) error {

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
