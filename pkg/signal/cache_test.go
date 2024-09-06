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
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

var cache *redisLib.MockedRedisStore

func loadModel(t *testing.T) model.Model {
	m := model.Model{}
	bytes, err := os.ReadFile("../model/test.yaml")
	assert.NoError(t, err)
	err = yaml.Unmarshal(bytes, &m)
	assert.NoError(t, err)
	return m
}

func Test_UpdateCellsCache(t *testing.T) {
	m := loadModel(t)
	ueHeight := 1.5
	updatedCells := map[string]*model.Cell{}
	for _, cell := range m.Cells {
		ncgi := strconv.FormatUint(uint64(cell.NCGI), 10)
		cellCopy := cell
		updatedCells[ncgi] = &cellCopy
	}
	assert.Equal(t, 3, len(updatedCells))
	cache = &redisLib.MockedRedisStore{}
	UpdateCells(updatedCells, cache, ueHeight, -87.0, 50, "1234")
	assert.Equal(t, 3, len(updatedCells))
	assert.Greater(t, len(updatedCells["17660905570307"].RPCoverageBoundaries[0].BoundaryPoints), 1000)
	assert.Greater(t, len(updatedCells["17660905553922"].RPCoverageBoundaries[0].BoundaryPoints), 1000)
	assert.Greater(t, len(updatedCells["17660905537537"].RPCoverageBoundaries[0].BoundaryPoints), 1000)

	assert.Greater(t, len(updatedCells["17660905570307"].CoverageBoundaries[0].BoundaryPoints), 100)
	assert.Greater(t, len(updatedCells["17660905553922"].CoverageBoundaries[0].BoundaryPoints), 100)
	assert.Greater(t, len(updatedCells["17660905537537"].CoverageBoundaries[0].BoundaryPoints), 100)

	assert.Greater(t, len(updatedCells["17660905570307"].Grid.GridPoints), 100)
	assert.Greater(t, len(updatedCells["17660905553922"].Grid.GridPoints), 100)
	assert.Greater(t, len(updatedCells["17660905537537"].Grid.GridPoints), 100)

	assert.Greater(t, len(updatedCells["17660905570307"].Grid.ShadowingMap), 100)
	assert.Greater(t, len(updatedCells["17660905553922"].Grid.ShadowingMap), 100)
	assert.Greater(t, len(updatedCells["17660905537537"].Grid.ShadowingMap), 100)

}

func Test_GenerateUEsLocations(t *testing.T) {
	m := loadModel(t)
	ueHeight := 1.5
	updatedCells := map[string]*model.Cell{}
	for _, cell := range m.Cells {
		ncgi := strconv.FormatUint(uint64(cell.NCGI), 10)
		cellCopy := cell
		updatedCells[ncgi] = &cellCopy
	}
	assert.Equal(t, 3, len(updatedCells))

	if cache == nil {
		cache = &redisLib.MockedRedisStore{}
	}
	UpdateCells(updatedCells, cache, ueHeight, -87.0, 50, "1234")

	uesLocations := make(map[uint64]map[int][]model.Coordinate)

	cellCqiUesMap := map[uint64]map[int]int{
		17660905537537: {1: 10, 5: 10, 10: 10, 15: 10},
		17660905570307: {1: 10, 5: 10, 10: 10, 15: 10},
		17660905553922: {1: 10, 5: 10, 10: 10, 15: 10},
	}

	for sCellNCGI, cqiMap := range cellCqiUesMap {

		if _, exists := uesLocations[sCellNCGI]; !exists {
			uesLocations[sCellNCGI] = make(map[int][]model.Coordinate)
		}
		for cqi, numUEs := range cqiMap {
			ueSINR := GetSINR(cqi)
			ueLocationForCqi := GenerateUEsLocations(sCellNCGI, numUEs, cqi, ueSINR, ueHeight, 50, updatedCells)
			assert.Equal(t, numUEs, len(ueLocationForCqi))
		}
	}
}
