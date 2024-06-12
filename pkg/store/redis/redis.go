package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/ran-simulator/pkg/model"
	"github.com/redis/go-redis/v9"
)

var log = logging.GetLogger()

func InitClient(redisHost string, redisPort string) *redis.Client {

	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: "",
		DB:       1,
	})
}

func AddCellSignalParams(rdb *redis.Client, ncgi uint64, ShadowMap *model.ShadowMap) error {

	if err := storeCellSignalParams(rdb, ncgi, ShadowMap); err != nil {
		return fmt.Errorf("failed to add cell signal params for cell %d: %v", ncgi, err)
	}
	log.Info("Added CellSignalParams for cell: %s", ncgi)

	return nil
}

func GetCellSignalParamsByNCGI(rdb *redis.Client, ncgi uint64) (*model.ShadowMap, error) {

	ncgiStr := strconv.FormatUint(ncgi, 10)
	ShadowMapData, err := rdb.HGetAll(context.Background(), ncgiStr).Result()
	if err != nil {
		return nil, fmt.Errorf("error fetching cell signal params data: %v", err)
	}

	if len(ShadowMapData) == 0 {
		return nil, fmt.Errorf("cell signal params with ncgi %s does not exist", ncgiStr)
	}

	shadowingMapBytes, exists := ShadowMapData["shadowingMap"]
	if !exists {
		return nil, fmt.Errorf("shadowingMap not found in cell signal params data")
	}

	gridPointsBytes, exists := ShadowMapData["gridPoints"]
	if !exists {
		return nil, fmt.Errorf("gridPoints not found in cell signal params data")
	}

	shadowingMap := [][]float64{{}}
	gridPoints := []model.Coordinate{}

	err = json.Unmarshal([]byte(shadowingMapBytes), &shadowingMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal shadowing map: %v ", err)
	}
	err = json.Unmarshal([]byte(gridPointsBytes), &gridPoints)
	if err != nil {
		log.Errorf("failed to unmarshal cell measurements: %v ", err)

		return nil, fmt.Errorf("failed to unmarshal grid points: %v ", err)
	}

	ShadowMap := &model.ShadowMap{
		ShadowingMap: shadowingMap,
		GridPoints:   gridPoints,
	}

	return ShadowMap, nil
}

func UpdateCellSignalParams(rdb *redis.Client, ncgi uint64, ShadowMap *model.ShadowMap) error {

	ShadowMapData, err := GetCellSignalParamsByNCGI(rdb, ncgi)
	if err != nil {
		return fmt.Errorf("error fetching cell signal params data: %v", err)
	}

	if len(ShadowMap.ShadowingMap) == 0 {
		ShadowMap.ShadowingMap = ShadowMapData.ShadowingMap
	}
	if len(ShadowMap.GridPoints) == 0 {
		ShadowMap.GridPoints = ShadowMapData.GridPoints
	}

	if err := storeCellSignalParams(rdb, ncgi, ShadowMap); err != nil {
		return fmt.Errorf("failed to update cell signal params for cell %d: %v", ncgi, err)
	}
	log.Info("Updated CellSignalParams for cell: %s", ncgi)

	return nil
}

func DeleteCellSignalParams(rdb *redis.Client, ncgi uint64) error {
	ncgiStr := strconv.FormatUint(ncgi, 10)

	err := rdb.Del(context.Background(), ncgiStr).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cell signal params for cell %s: %v", ncgiStr, err)
	}
	log.Infof("Deleted CellSignalParams for cell: %s", ncgiStr)
	return nil
}

func storeCellSignalParams(rdb *redis.Client, ncgi uint64, ShadowMap *model.ShadowMap) error {
	ncgiStr := strconv.FormatUint(ncgi, 10)

	shadowingMapBytes, err := json.Marshal(ShadowMap.ShadowingMap)
	if err != nil {
		return fmt.Errorf("failed to marshal simulator model: %v ", err)
	}

	gridPointsBytes, err := json.Marshal(ShadowMap.GridPoints)
	if err != nil {
		return fmt.Errorf("failed to marshal cell measurements: %v ", err)
	}

	ShadowMapMap := map[string]interface{}{
		"shadowingMap": shadowingMapBytes,
		"gridPoints":   gridPointsBytes,
	}

	return rdb.HSet(context.Background(), ncgiStr, ShadowMapMap).Err()
}
