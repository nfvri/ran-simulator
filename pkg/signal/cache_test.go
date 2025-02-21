// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package signal

import (
	"os"
	"strconv"
	"testing"

	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
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
	beamID1 := model.BeamID{NCGI: 17660905570307, CarrierIndex: 1, BeamIndex: 1}
	beamID2 := model.BeamID{NCGI: 17660905553922, CarrierIndex: 1, BeamIndex: 1}
	beamID3 := model.BeamID{NCGI: 17660905537537, CarrierIndex: 1, BeamIndex: 1}
	assert.Greater(t, len(m.Cells["17660905570307"].CachedStates[m.Cells["17660905570307"].CurrentStateHash].RPCoverageBoundaries[beamID1][0].BoundaryPoints), 1000)
	assert.Greater(t, len(m.Cells["17660905553922"].CachedStates[m.Cells["17660905553922"].CurrentStateHash].RPCoverageBoundaries[beamID2][0].BoundaryPoints), 1000)
	assert.Greater(t, len(m.Cells["17660905537537"].CachedStates[m.Cells["17660905537537"].CurrentStateHash].RPCoverageBoundaries[beamID3][0].BoundaryPoints), 1000)

	assert.Greater(t, len(m.Cells["17660905570307"].CachedStates[m.Cells["17660905570307"].CurrentStateHash].CoverageBoundaries[beamID1][0].BoundaryPoints), 100)
	assert.Greater(t, len(m.Cells["17660905553922"].CachedStates[m.Cells["17660905553922"].CurrentStateHash].CoverageBoundaries[beamID2][0].BoundaryPoints), 100)
	assert.Greater(t, len(m.Cells["17660905537537"].CachedStates[m.Cells["17660905537537"].CurrentStateHash].CoverageBoundaries[beamID3][0].BoundaryPoints), 100)

	assert.Greater(t, len(m.Cells["17660905570307"].Grid.GridPoints), 100)
	assert.Greater(t, len(m.Cells["17660905553922"].Grid.GridPoints), 100)
	assert.Greater(t, len(m.Cells["17660905537537"].Grid.GridPoints), 100)

	assert.Greater(t, len(m.Cells["17660905570307"].Grid.ShadowingMaps), 100)
	assert.Greater(t, len(m.Cells["17660905553922"].Grid.ShadowingMaps), 100)
	assert.Greater(t, len(m.Cells["17660905537537"].Grid.ShadowingMaps), 100)

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

		numUEsPerBeamQS := bw.GetNumUEsPerBeamQS(sCell, cqiMap)
		nCells := utils.GetNeighborCells(sCell, m.Cells, utils.By.Freq)
		nBeamIDs := GetNeighborBeamIDs(nCells)

		if _, exists := uesLocations[sCellNCGI]; !exists {
			uesLocations[sCellNCGI] = make(map[int][]model.Coordinate)
		}
		for beamQS, numUEs := range numUEsPerBeamQS {
			ueSINR := GetSINR(beamQS.CQI)

			ueLocationForCqi := GetSinrPoints(sCell, beamQS.BeamID, nCells, nBeamIDs, ueHeight, ueSINR, 200, numUEs, beamQS.CQI)
			assert.Equal(t, numUEs, len(ueLocationForCqi))
		}
	}
}
