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

func AddCellSignalParams(rdb *redis.Client, ncgi uint64, cellSignalParams *model.CellSignalParams) error {

	if err := storeCellSignalParams(rdb, ncgi, cellSignalParams); err != nil {
		return fmt.Errorf("failed to add cell signal params for cell %d: %v", ncgi, err)
	}
	log.Info("Added CellSignalParams for cell: %s", ncgi)

	return nil
}

func GetCellSignalParamsByNCGI(rdb *redis.Client, ncgi uint64) (*model.CellSignalParams, error) {

	ncgiStr := strconv.FormatUint(ncgi, 10)
	cellSignalParamsData, err := rdb.HGetAll(context.Background(), ncgiStr).Result()
	if err != nil {
		return nil, fmt.Errorf("error fetching cell signal params data: %v", err)
	}

	if len(cellSignalParamsData) == 0 {
		return nil, fmt.Errorf("cell signal params with ncgi %s does not exist", ncgiStr)
	}

	shadowingMapBytes, exists := cellSignalParamsData["shadowingMap"]
	if !exists {
		return nil, fmt.Errorf("shadowingMap not found in cell signal params data")
	}

	gridPointsBytes, exists := cellSignalParamsData["gridPoints"]
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

	cellSignalParams := &model.CellSignalParams{
		ShadowingMap: shadowingMap,
		GridPoints:   gridPoints,
	}

	return cellSignalParams, nil
}

func UpdateCellSignalParams(rdb *redis.Client, ncgi uint64, cellSignalParams *model.CellSignalParams) error {

	cellSignalParamsData, err := GetCellSignalParamsByNCGI(rdb, ncgi)
	if err != nil {
		return fmt.Errorf("error fetching cell signal params data: %v", err)
	}

	if len(cellSignalParams.ShadowingMap) == 0 {
		cellSignalParams.ShadowingMap = cellSignalParamsData.ShadowingMap
	}
	if len(cellSignalParams.GridPoints) == 0 {
		cellSignalParams.GridPoints = cellSignalParamsData.GridPoints
	}

	if err := storeCellSignalParams(rdb, ncgi, cellSignalParams); err != nil {
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

func storeCellSignalParams(rdb *redis.Client, ncgi uint64, cellSignalParams *model.CellSignalParams) error {
	ncgiStr := strconv.FormatUint(ncgi, 10)

	shadowingMapBytes, err := json.Marshal(cellSignalParams.ShadowingMap)
	if err != nil {
		return fmt.Errorf("failed to marshal simulator model: %v ", err)
	}

	gridPointsBytes, err := json.Marshal(cellSignalParams.GridPoints)
	if err != nil {
		return fmt.Errorf("failed to marshal cell measurements: %v ", err)
	}

	cellSignalParamsMap := map[string]interface{}{
		"shadowingMap": shadowingMapBytes,
		"gridPoints":   gridPointsBytes,
	}

	return rdb.HSet(context.Background(), ncgiStr, cellSignalParamsMap).Err()
}
