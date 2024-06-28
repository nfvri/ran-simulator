package cells

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

type RedisStore struct {
	Rdb *redis.Client
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

func (s *RedisStore) Add(ctx context.Context, cell *model.Cell) error {
	ncgiStr := strconv.FormatUint(uint64(cell.NCGI), 10)

	cellBytes, err := json.Marshal(cell)
	if err != nil {
		return fmt.Errorf("failed to marshal simulator model: %v ", err)
	}

	return s.Rdb.Set(context.Background(), ncgiStr, cellBytes, time.Duration(0)).Err()
}

func (s *RedisStore) Get(ctx context.Context, ncgi types.NCGI) (*model.Cell, error) {
	ncgiStr := strconv.FormatUint(uint64(ncgi), 10)
	cellBytes, err := s.Rdb.Get(context.Background(), ncgiStr).Result()
	if err != nil {
		return nil, fmt.Errorf("error fetching cell signal params data: %v", err)
	}

	if len(cellBytes) == 0 {
		return nil, fmt.Errorf("cell signal params with ncgi %s does not exist", ncgiStr)
	}

	cell := model.Cell{}

	err = json.Unmarshal([]byte(cellBytes), &cell)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal shadowing map: %v ", err)
	}

	return &cell, nil
}

func (s *RedisStore) Update(ctx context.Context, cell *model.Cell) error {
	return s.Add(ctx, cell)
}

func (s *RedisStore) Delete(ctx context.Context, ncgi types.NCGI) (*model.Cell, error) {
	cell, err := s.Get(ctx, ncgi)
	if err != nil {
		return nil, err
	}
	ncgiStr := strconv.FormatUint(uint64(ncgi), 10)
	err = s.Rdb.Del(ctx, ncgiStr).Err()
	return cell, err
}
