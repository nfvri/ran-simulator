// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

//go:build !race
// +build !race

package mobility

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/nfvri/ran-simulator/pkg/handover"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/store/cells"
	"github.com/nfvri/ran-simulator/pkg/store/event"
	"github.com/nfvri/ran-simulator/pkg/store/nodes"
	"github.com/nfvri/ran-simulator/pkg/store/routes"
	"github.com/nfvri/ran-simulator/pkg/store/ues"
	"github.com/stretchr/testify/assert"
)

func TestDriver(t *testing.T) {
	m := &model.Model{}
	err := model.LoadConfig(m, "../model/test")
	assert.NoError(t, err)
	ctx := context.Background()
	ns := nodes.NewNodeRegistry(ctx, m.Nodes)
	cs := cells.NewCellRegistry(ctx, m.Cells, ns)
	us := ues.NewUERegistry(ctx, m, cs, "random")
	rs := routes.NewRouteRegistry()

	hoHandler := handover.NewA3HandoverHandler(m)
	ho := handover.NewA3Handover(hoHandler)
	hoCtrl := handover.NewHOController(handover.A3, ho)

	ctxTODO := context.TODO()
	ch := make(chan event.Event)
	err = us.Watch(ctxTODO, ch, ues.WatchOptions{Replay: true})
	assert.NoError(t, err)

	e := <-ch
	ue := e.Value.(model.UE)

	route := &model.Route{
		IMSI:     ue.IMSI,
		Points:   []*model.Coordinate{{Lat: 50.001, Lng: 0.0000}, {Lat: 50.0000, Lng: 0.0000}, {Lat: 50.0000, Lng: 0.0002}},
		SpeedAvg: 40000.0,
	}
	err = rs.Add(ctxTODO, route)
	assert.NoError(t, err)
	finishHOsChan := make(chan bool)
	driver := NewMobilityDriver(m, "local", hoCtrl, finishHOsChan)
	driver.Start(ctxTODO)

	c := 0
	for e = range ch {
		ue = e.Value.(model.UE)
		fmt.Printf("%v: %v\n", ue.Location, ue.Heading)
		c = c + 1
		if c == 2 {
			assert.True(t, 52.35 < ue.Location.Lat && ue.Location.Lat < 52.60, "UE latitude is out of range")
			assert.True(t, 13.25 < ue.Location.Lng && ue.Location.Lng < 13.55, "UE longitude is out of range")
			assert.Equal(t, uint32(0), ue.Heading)

		} else if c == 6 {
			assert.Equal(t, uint32(0), ue.Heading)
			break
		}
	}

	defer close(finishHOsChan)
	for range finishHOsChan {
		fmt.Print("HOs completed")
		return
	}
	driver.Stop()

}

func TestRouteGeneration(t *testing.T) {
	m := &model.Model{}
	err := model.LoadConfig(m, "../utils/honeycomb/sample")
	assert.NoError(t, err)

	ctx := context.Background()

	ns := nodes.NewNodeRegistry(ctx, m.Nodes)
	cs := cells.NewCellRegistry(ctx, m.Cells, ns)
	us := ues.NewUERegistry(ctx, m, cs, "random")

	ctxTODO := context.TODO()
	us.SetUECount(ctxTODO, 100)
	assert.Equal(t, 100, us.Len(ctxTODO))
	storeUes := us.ListAllUEs(ctxTODO)
	m.UEs = make(map[string]*model.UE)
	for _, ue := range storeUes {
		ueCopy := *ue
		imsiStr := strconv.Itoa(int(ue.IMSI))
		m.UEs[imsiStr] = &ueCopy
	}
	hoHandler := handover.NewA3HandoverHandler(m)
	ho := handover.NewA3Handover(hoHandler)
	hoCtrl := handover.NewHOController(handover.A3, ho)

	finishHOsChan := make(chan bool)
	driver := NewMobilityDriver(m, "local", hoCtrl, finishHOsChan)
	driver.GenerateRoutes(ctxTODO, 30000, 160000, 20000, nil, false)

	ch := make(chan event.Event)
	err = us.Watch(ctxTODO, ch, ues.WatchOptions{Replay: true})
	assert.NoError(t, err)

	driver.Start(ctxTODO)

	c := 0
	for e := range ch {
		ue := e.Value.(model.UE)
		assert.True(t, 52.35 < ue.Location.Lat && ue.Location.Lat < 52.60, "UE latitude is out of range")
		assert.True(t, 13.25 < ue.Location.Lng && ue.Location.Lng < 13.55, "UE longitude is out of range")
		c = c + 1
		if c > 99 {
			break
		}
	}

	driver.Stop()
}
