// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package ues

import (
	"context"
	"math/rand"
	"os"
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/store/cells"
	"github.com/nfvri/ran-simulator/pkg/store/nodes"
	redisLib "github.com/nfvri/ran-simulator/pkg/store/redis"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	"gopkg.in/yaml.v2"

	"github.com/stretchr/testify/assert"
)

func loadModel(t *testing.T) model.Model {
	m := model.Model{}
	bytes, err := os.ReadFile("../../model/test.yaml")
	assert.NoError(t, err)
	err = yaml.Unmarshal(bytes, &m)
	assert.NoError(t, err)
	t.Log(m)
	return m
}

func cellStore(m model.Model) cells.Store {
	return cells.NewCellRegistry(m.Cells, nodes.NewNodeRegistry(m.Nodes))
}

func TestUERegistry(t *testing.T) {

	m := loadModel(t)
	ctx := context.Background()
	ues := NewUERegistry(&m, cellStore(m), &redisLib.MockedRedisStore{}, "random")
	assert.NotNil(t, ues, "unable to create UE registry")
	assert.Equal(t, 12, ues.Len(ctx))

	ues.SetUECount(ctx, 10)
	assert.Equal(t, 10, ues.Len(ctx))

	ues.SetUECount(ctx, 200)
	assert.Equal(t, 200, ues.Len(ctx))
}

func TestMoveUEsToCell(t *testing.T) {
	m := loadModel(t)
	ctx := context.Background()
	cellStore := cellStore(m)
	ues := NewUERegistry(&m, cellStore, &redisLib.MockedRedisStore{}, "random")
	assert.NotNil(t, ues, "unable to create UE registry")
	// Get a cell NCGI
	cell1, err := cellStore.GetRandomCell()
	assert.NoError(t, err)
	ecgi1 := cell1.NCGI

	// Get another cell NCGI; make sure it's different than the first.
	cell2, err := cellStore.GetRandomCell()
	assert.NoError(t, err)
	ecgi2 := cell2.NCGI
	for ecgi1 == ecgi2 {
		cell2, err = cellStore.GetRandomCell()
		assert.NoError(t, err)
		ecgi2 = cell2.NCGI
	}

	for i, ue := range ues.ListAllUEs(ctx) {
		ncgi := ecgi1
		if i%3 == 0 {
			ncgi = ecgi2
		}
		err := ues.MoveToCell(ctx, ue.IMSI, ncgi, rand.Float64())
		assert.NoError(t, err)
	}

	assert.Equal(t, 8, len(ues.ListUEs(ctx, ecgi1)))
	assert.Equal(t, 4, len(ues.ListUEs(ctx, ecgi2)))
}

func TestMoveUEToCell(t *testing.T) {
	m := loadModel(t)
	ctx := context.Background()
	cellStore := cellStore(m)
	ues := NewUERegistry(&m, cellStore, &redisLib.MockedRedisStore{}, "random")
	assert.NotNil(t, ues, "unable to create UE registry")
	ue := ues.ListAllUEs(ctx)[0]
	err := ues.MoveToCell(ctx, ue.IMSI, types.NCGI(321), 11.0)
	assert.NoError(t, err)
	ue1, _ := ues.Get(ctx, ue.IMSI)
	assert.NoError(t, err)
	assert.Equal(t, types.NCGI(321), ue1.Cell.NCGI)
	assert.Equal(t, 11.0, ue1.Cell.Rsrp)
	list := ues.ListAllUEs(ctx)
	assert.Len(t, list, 12)
	for _, ue := range list {
		if ue.Cell.NCGI == types.NCGI(321) {
			return
		}
	}
	assert.Fail(t, "boom")
}

func TestMoveUEToCoord(t *testing.T) {
	m := loadModel(t)
	ctx := context.Background()
	cellStore := cellStore(m)
	ues := NewUERegistry(&m, cellStore, &redisLib.MockedRedisStore{}, "random")
	assert.NotNil(t, ues, "unable to create UE registry")

	ue := ues.ListAllUEs(ctx)[0]
	err := ues.MoveToCoordinate(ctx, ue.IMSI, model.Coordinate{Lat: 50.0755, Lng: 14.4378}, 182)
	assert.NoError(t, err)

	ue1, _ := ues.Get(ctx, ue.IMSI)
	assert.NoError(t, err)
	assert.Equal(t, 50.0755, ue1.Location.Lat)
	assert.Equal(t, 14.4378, ue1.Location.Lng)
	assert.Equal(t, uint32(182), ue1.Heading)
}

func TestUpdateCells(t *testing.T) {
	m := loadModel(t)
	ctx := context.Background()
	cellStore := cellStore(m)
	ues := NewUERegistry(&m, cellStore, &redisLib.MockedRedisStore{}, "random")
	assert.NotNil(t, ues, "unable to create UE registry")

	ue := ues.ListAllUEs(ctx)[0]
	uecells := []*model.UECell{{NCGI: 123001, Rsrp: 42.0}, {NCGI: 123002, Rsrp: 6.28}}
	err := ues.UpdateCells(ctx, ue.IMSI, uecells)
	assert.NoError(t, err)

	ue1, _ := ues.Get(ctx, ue.IMSI)
	assert.NoError(t, err)
	assert.Equal(t, 42.0, ue1.Cells[0].Rsrp)
	assert.Equal(t, 6.28, ue1.Cells[1].Rsrp)
}
