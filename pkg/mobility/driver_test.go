// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

//go:build !race
// +build !race

package mobility

import (
	"context"
	"fmt"
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/store/cells"
	"github.com/nfvri/ran-simulator/pkg/store/event"
	"github.com/nfvri/ran-simulator/pkg/store/nodes"
	redisLib "github.com/nfvri/ran-simulator/pkg/store/redis"
	"github.com/nfvri/ran-simulator/pkg/store/routes"
	"github.com/nfvri/ran-simulator/pkg/store/ues"
	"github.com/stretchr/testify/assert"
)

func TestDriver(t *testing.T) {
	m := &model.Model{}
	err := model.LoadConfig(m, "../model/test")
	assert.NoError(t, err)

	ns := nodes.NewNodeRegistry(m.Nodes)
	cs := cells.NewCellRegistry(m.Cells, ns)
	us := ues.NewUERegistry(m, cs, &redisLib.MockedRedisStore{}, "random")
	rs := routes.NewRouteRegistry()

	ctx := context.TODO()
	ch := make(chan event.Event)
	err = us.Watch(ctx, ch, ues.WatchOptions{Replay: true})
	assert.NoError(t, err)

	e := <-ch
	ue := e.Value.(*model.UE)

	route := &model.Route{
		IMSI:     ue.IMSI,
		Points:   []*model.Coordinate{{Lat: 50.001, Lng: 0.0000}, {Lat: 50.0000, Lng: 0.0000}, {Lat: 50.0000, Lng: 0.0002}},
		SpeedAvg: 40000.0,
	}
	err = rs.Add(ctx, route)
	assert.NoError(t, err)

	driver := NewMobilityDriver(m, "local", nil, nil)
	driver.Start(ctx)

	c := 0
	for e = range ch {
		ue = e.Value.(*model.UE)
		fmt.Printf("%v: %v\n", ue.Location, ue.Heading)
		c = c + 1
		if c == 2 {
			assert.Equal(t, 50.001, ue.Location.Lat)
			assert.Equal(t, 0.0, ue.Location.Lng)
			assert.Equal(t, uint32(180), ue.Heading)

		} else if c == 6 {
			assert.Equal(t, uint32(0), ue.Heading)
			break
		}
	}

	driver.Stop()
}

func TestRouteGeneration(t *testing.T) {
	m := &model.Model{}
	err := model.LoadConfig(m, "../utils/honeycomb/sample")
	assert.NoError(t, err)

	ns := nodes.NewNodeRegistry(m.Nodes)
	cs := cells.NewCellRegistry(m.Cells, ns)
	us := ues.NewUERegistry(m, cs, &redisLib.MockedRedisStore{}, "random")
	rs := routes.NewRouteRegistry()

	ctx := context.TODO()
	us.SetUECount(ctx, 100)
	assert.Equal(t, 100, us.Len(ctx))

	driver := NewMobilityDriver(m, "local", nil, nil)
	driver.GenerateRoutes(ctx, 30000, 160000, 20000, nil, false)
	assert.Equal(t, 100, rs.Len(ctx))

	ch := make(chan event.Event)
	err = us.Watch(ctx, ch, ues.WatchOptions{Replay: true})
	assert.NoError(t, err)

	driver.Start(ctx)

	c := 0
	for e := range ch {
		ue := e.Value.(*model.UE)
		fmt.Printf("%v: %v\n", ue.Location, ue.Heading)
		assert.True(t, 52.35 < ue.Location.Lat && ue.Location.Lat < 52.60, "UE latitude is out of range")
		assert.True(t, 13.25 < ue.Location.Lng && ue.Location.Lng < 13.55, "UE longitude is out of range")
		c = c + 1
		if c > 500 {
			break
		}
	}

	driver.Stop()
}
