// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package signal

import (
	"os"
	"strconv"
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	redisLib "github.com/nfvri/ran-simulator/pkg/store/redis"
	"github.com/nfvri/ran-simulator/pkg/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

var cache *redisLib.MockedRedisStore

func loadModel(t *testing.T) *model.Model {
	m := &model.Model{}
	bytes, err := os.ReadFile("../model/test.yaml")
	assert.NoError(t, err)
	err = yaml.Unmarshal(bytes, m)
	assert.NoError(t, err)
	return m
}

func Test_UpdateCellsCache(t *testing.T) {
	m := loadModel(t)
	ueHeight := 1.5

	assert.Equal(t, 3, len(m.Cells))
	cache = &redisLib.MockedRedisStore{}
	UpdateCells(m.Cells, cache, ueHeight, -87.0, 50, "1234")
	assert.Equal(t, 3, len(m.Cells))
	assert.Greater(t, len(m.Cells["17660905570307"].CachedStates[m.Cells["17660905570307"].CurrentStateHash].RPCoverageBoundaries[0].BoundaryPoints), 1000)
	assert.Greater(t, len(m.Cells["17660905553922"].CachedStates[m.Cells["17660905553922"].CurrentStateHash].RPCoverageBoundaries[0].BoundaryPoints), 1000)
	assert.Greater(t, len(m.Cells["17660905537537"].CachedStates[m.Cells["17660905537537"].CurrentStateHash].RPCoverageBoundaries[0].BoundaryPoints), 1000)

	assert.Greater(t, len(m.Cells["17660905570307"].CachedStates[m.Cells["17660905570307"].CurrentStateHash].CoverageBoundaries[0].BoundaryPoints), 100)
	assert.Greater(t, len(m.Cells["17660905553922"].CachedStates[m.Cells["17660905553922"].CurrentStateHash].CoverageBoundaries[0].BoundaryPoints), 100)
	assert.Greater(t, len(m.Cells["17660905537537"].CachedStates[m.Cells["17660905537537"].CurrentStateHash].CoverageBoundaries[0].BoundaryPoints), 100)

	assert.Greater(t, len(m.Cells["17660905570307"].Grid.GridPoints), 100)
	assert.Greater(t, len(m.Cells["17660905553922"].Grid.GridPoints), 100)
	assert.Greater(t, len(m.Cells["17660905537537"].Grid.GridPoints), 100)

	assert.Greater(t, len(m.Cells["17660905570307"].Grid.ShadowingMap), 100)
	assert.Greater(t, len(m.Cells["17660905553922"].Grid.ShadowingMap), 100)
	assert.Greater(t, len(m.Cells["17660905537537"].Grid.ShadowingMap), 100)

}

func Test_GenerateUEsLocations(t *testing.T) {
	m := loadModel(t)
	ueHeight := 1.5
	assert.Equal(t, 3, len(m.Cells))

	if cache == nil {
		cache = &redisLib.MockedRedisStore{}
	}
	UpdateCells(m.Cells, cache, ueHeight, -87.0, 50, "1234")

	uesLocations := make(map[uint64]map[int][]model.Coordinate)

	cellCqiUesMap := map[uint64]map[int]int{
		17660905537537: {1: 10, 5: 10, 10: 10, 15: 10},
		17660905570307: {1: 10, 5: 10, 10: 10, 15: 10},
		17660905553922: {1: 10, 5: 10, 10: 10, 15: 10},
	}

	for sCellNCGI, cqiMap := range cellCqiUesMap {
		sCell, ok := m.Cells[strconv.FormatUint(sCellNCGI, 10)]
		if !ok {
			continue
		}
		if _, exists := uesLocations[sCellNCGI]; !exists {
			uesLocations[sCellNCGI] = make(map[int][]model.Coordinate)
		}
		for cqi, numUEs := range cqiMap {
			ueSINR := GetSINR(cqi)
			neighborCells := utils.GetNeighborCells(sCell, m.Cells)
			ueLocationForCqi := GetSinrPoints(ueHeight, sCell, neighborCells, ueSINR, 200, numUEs, cqi)
			assert.Equal(t, numUEs, len(ueLocationForCqi))
		}
	}
}
