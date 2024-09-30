// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package mobility

import (
	"context"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"

	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"
	"github.com/sirupsen/logrus"

	"github.com/nfvri/ran-simulator/pkg/handover"
	"github.com/nfvri/ran-simulator/pkg/measurement"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/signal"
	"github.com/nfvri/ran-simulator/pkg/store/cells"
	"github.com/nfvri/ran-simulator/pkg/store/routes"
	"github.com/nfvri/ran-simulator/pkg/store/ues"
	"github.com/nfvri/ran-simulator/pkg/utils"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger()

// Driver is an abstraction of an entity driving the UE mobility
type Driver interface {
	// Start starts the driving engine
	Start(ctx context.Context)

	// Stop stops the driving engine
	Stop()

	// GenerateRoutes generates routes for all UEs that currently do not have a route; remove routes with no UEs
	GenerateRoutes(ctx context.Context, minSpeed uint32, maxSpeed uint32, speedStdDev uint32, routeEndPoints []model.RouteEndPoint, directRoute bool)

	// GetMeasCtrl returns the Measurement Controller
	GetMeasCtrl() measurement.MeasController

	// GetHoCtrl
	GetHoCtrl() handover.HOController

	// GetRrcCtrl returns the Rrc Controller
	GetRrcCtrl() RrcCtrl

	// Handover
	Handover(ctx context.Context, ue *model.UE, tCell *model.UECell)

	//GetHoLogic
	GetHoLogic() string

	//SetHoLogic
	SetHoLogic(hoLogic string)

	// AddRrcChan
	AddRrcChan(ch chan model.UE)

	UpdateUESignalStrength(imsi types.IMSI)
}

type driver struct {
	m                       *model.Model
	cellStore               cells.Store
	routeStore              routes.Store
	ueStore                 ues.Store
	apiKey                  string
	ticker                  *time.Ticker
	done                    chan bool
	stopLocalHO             chan bool
	min                     *model.Coordinate
	max                     *model.Coordinate
	measCtrl                measurement.MeasController
	hoCtrl                  handover.HOController
	hoLogic                 string
	rrcCtrl                 RrcCtrl
	ueLock                  map[types.IMSI]*sync.Mutex
	rrcStateChangesDisabled bool
	wayPointRoute           bool
	finishHOsChan           chan bool
}

// NewMobilityDriver returns a driving engine capable of "driving" UEs along pre-specified routes
func NewMobilityDriver(m *model.Model, hoLogic string, hoCtrl handover.HOController, finishHOsChan chan bool) Driver {
	return &driver{
		m:                       m,
		hoLogic:                 hoLogic,
		rrcCtrl:                 NewRrcCtrl(m.UECountPerCell),
		rrcStateChangesDisabled: m.RrcStateChangesDisabled,
		wayPointRoute:           m.WayPointRoute,
		hoCtrl:                  hoCtrl,
		finishHOsChan:           finishHOsChan,
	}
}

// var tickUnit = time.Second
// const measType = "EventA3" // ToDo: should be programmable

const tickFrequency = 1

func (d *driver) Start(ctx context.Context) {
	log.Info("Driver starting")

	// // Iterate over all routes and position the UEs at the start of their routes
	// for _, route := range d.routeStore.List(ctx) {
	// 	d.initializeUEPosition(ctx, route)
	// }

	d.ueLock = make(map[types.IMSI]*sync.Mutex)
	for _, ue := range d.m.UEList {
		d.ueLock[ue.IMSI] = &sync.Mutex{}
	}

	// d.cellLock = make(map[types.NCGI]*sync.Mutex)
	// for _, cell := range d.m.Cells {
	// 	d.cellLock[cell.NCGI] = &sync.Mutex{}
	// }

	// d.ticker = time.NewTicker(tickFrequency * tickUnit)
	// d.done = make(chan bool)
	d.stopLocalHO = make(chan bool)

	// Add measController
	// d.measCtrl = measurement.NewMeasController(measType, d.cellStore, d.ueStore)
	// d.measCtrl.Start(ctx)
	d.hoCtrl.Start()
	// link measController with hoController
	// go d.linkMeasCtrlHoCtrl()

	// Add hoController
	if d.hoLogic == "local" {
		log.Info("HO logic is running locally")
		// process handover decision
		go d.processHandoverDecision(ctx)
	} else if d.hoLogic == "mho" {
		log.Info("HO logic is running outside - mho")
	} else {
		log.Warn("There is no handover logic - running measurement only")
	}

	// go d.drive(ctx)
}

func (d *driver) Stop() {
	log.Info("Driver stopping")
	// d.ticker.Stop()
	// d.done <- true
	d.stopLocalHO <- true
}

func (d *driver) GetMeasCtrl() measurement.MeasController {
	return d.measCtrl
}

func (d *driver) GetHoCtrl() handover.HOController {
	return d.hoCtrl
}

func (d *driver) GetRrcCtrl() RrcCtrl {
	return d.rrcCtrl
}

func (d *driver) SetHoLogic(hoLogic string) {
	if d.hoLogic == "local" && hoLogic == "mho" {
		log.Info("Stopping local HO")
		d.stopLocalHO <- true
	} else if d.hoLogic == "mho" && hoLogic == "local" {
		log.Info("Starting local HO")
		go d.linkMeasCtrlHoCtrl()
	}
	d.hoLogic = hoLogic
}

func (d *driver) AddRrcChan(ch chan model.UE) {
	d.addRrcChan(ch)
}

func (d *driver) lockUE(imsi types.IMSI) {
	d.ueLock[imsi].Lock()
}

func (d *driver) unlockUE(imsi types.IMSI) {
	if _, ok := d.ueLock[imsi]; !ok {
		log.Errorf("lock not found for IMSI %d", imsi)
		return
	}
	d.ueLock[imsi].Unlock()
}

// func (d *driver) lockCell(ncgi types.NCGI) {
// 	d.cellLock[ncgi].Lock()
// }

// func (d *driver) unlockCell(ncgi types.NCGI) {
// 	if _, ok := d.cellLock[ncgi]; !ok {
// 		log.Errorf("lock not found for ncgi %d", ncgi)
// 		return
// 	}
// 	d.cellLock[ncgi].Unlock()
// }

func (d *driver) Drive(ctx context.Context) {
	for {
		select {
		case <-d.done:
			ctx.Done()
			close(d.done)
			return
		case <-d.ticker.C:
			for _, route := range d.routeStore.List(ctx) {
				go d.processRoute(ctx, route)

			}
		}
	}
}

func (d *driver) processRoute(ctx context.Context, route *model.Route) {
	d.lockUE(route.IMSI)
	defer d.unlockUE(route.IMSI)
	if route.NextPoint == 0 && !route.Reverse {
		d.initializeUEPosition(ctx, route)
	}
	d.updateUEPosition(ctx, route)
	d.UpdateUESignalStrength(route.IMSI)
	if !d.rrcStateChangesDisabled {
		d.updateRrc(ctx, route.IMSI)
	}
	d.updateFiveQI(ctx, route.IMSI)
	d.reportMeasurement(route.IMSI)
}

// Initializes UE positions to the start of its routes.
func (d *driver) initializeUEPosition(ctx context.Context, route *model.Route) {
	// bearing := utils.InitialBearing(*route.Points[0], *route.Points[1])
	// _ = d.ueStore.MoveToCoordinate(ctx, route.IMSI, *route.Points[0], uint32(math.Round(bearing)))
	// _ = d.routeStore.Start(ctx, route.IMSI, route.SpeedAvg, route.SpeedStdDev)
}

func (d *driver) updateUEPosition(ctx context.Context, route *model.Route) {
	// Get the UE
	ue, ok := d.m.UEList[strconv.FormatUint(uint64(route.IMSI), 10)]
	if !ok {
		log.Warn("Unable to find UE %d", route.IMSI)
		return
	}

	// Determine speed and heading
	speed := float64(route.SpeedAvg) + rand.NormFloat64()*float64(route.SpeedStdDev)
	distanceDriven := (tickFrequency * speed) / 3600.0

	// Determine bearing and distance to the next point
	// bearing := utils.InitialBearing(ue.Location, *route.Points[route.NextPoint])
	remainingDistance := utils.Distance(ue.Location, *route.Points[route.NextPoint])

	// If distance is less than to the next waypoint, determine the coordinate along that vector
	// Otherwise just use the next waypoint
	// newPoint := *route.Points[route.NextPoint]
	reachedWaypoint := remainingDistance <= distanceDriven
	if d.wayPointRoute {
		reachedWaypoint = true
	}
	// if !reachedWaypoint {
	// 	newPoint = utils.TargetPoint(ue.Location, bearing, distanceDriven)
	// }

	// Move the UE to the determined coordinate; update heading if necessary
	// err = d.ueStore.MoveToCoordinate(ctx, route.IMSI, newPoint, uint32(math.Round(bearing)))
	// if err != nil {
	// 	log.Warn("Unable to update UE %d coordinates", route.IMSI)
	// }

	// Update the route if necessary
	if reachedWaypoint {
		_ = d.routeStore.Advance(ctx, route.IMSI)
	}
}

func (d *driver) reportMeasurement(imsi types.IMSI) {
	ue, ok := d.m.UEList[strconv.FormatUint(uint64(imsi), 10)]
	if !ok {
		log.Warn("Unable to find UE %d", imsi)
		return
	}

	// Skip reporting measurement for IDLE UE
	if ue.RrcState == e2sm_mho.Rrcstatus_RRCSTATUS_IDLE {
		return
	}

	d.measCtrl.GetInputChan() <- &ue
}

func (d *driver) linkMeasCtrlHoCtrl() {
	log.Info("Connecting measurement and handover controllers")
	// for report := range d.measCtrl.GetOutputChan() {
	// 	d.hoCtrl.GetInputChan() <- report
	// }
}

func (d *driver) processHandoverDecision(ctx context.Context) {
	log.Info("Handover decision process starting")
	for {
		select {
		case hoDecision := <-d.hoCtrl.GetOutputChan():
			log.Debugf("Received HO Decision: %v", hoDecision)
			if hoDecision.Feasible {
				d.Handover(ctx, hoDecision.UE, hoDecision.TargetCell)
			} else {
				// TODO: delete ue.BwpRefs and sCell.BWPs from cell but not transfer them and ue.Rrcstatus_RRCSTATUS_INACTIVE
				logrus.Warnf("Handover infeasible for imsi: %v, at cell: %v",
					hoDecision.UE.IMSI,
					hoDecision.TargetCell.NCGI,
				)
			}
		case <-d.stopLocalHO:
			log.Info("local HO stopped")
			d.finishHOsChan <- true
			return
		}
	}
}

// Handover handovers ue to target cell
func (d *driver) Handover(ctx context.Context, ue *model.UE, tCell *model.UECell) {
	log.Infof("Handover() imsi:%v, tCell:%v", ue.IMSI, tCell)
	// d.lockUE(ue.IMSI)
	// defer d.unlockUE(ue.IMSI)

	// Update RRC state on handover
	if ue.Cell.NCGI == tCell.NCGI {
		log.Infof("Duplicate HO skipped imsi%d, cgi:%v", ue.IMSI, tCell.NCGI)
		return
	}

	if ue.RrcState != e2sm_mho.Rrcstatus_RRCSTATUS_CONNECTED {
		log.Warnf("HO skipped for not connected UE %d", ue.IMSI)
		return
	}

	// d.DecrementRrcConnectedCount(ue.Cell.NCGI)
	// d.IncrementRrcConnectedCount(tCell.NCGI)

	//TODO: update ueCell.BwpRefs and delete oldcell.BWPs and append tCell.BWPs and tueCell.BWPs
	// ue.Cell <- new cell
	// append(ue.Cells, oldSCell)
	// d.UpdateBWPRefs()

	// after changing serving cell, calculate channel quality/signal strength again

	// TODO: update all ueCells metrics(rsrp,rsrq,sinr) for sCell and cCells
	d.UpdateUESignalStrength(ue.IMSI)
	// d.ueStore.UpdateMaxUEsPerCell(ctx)

	log.Infof("HO is done successfully: %v to %v", ue.IMSI, tCell)
}

func (d *driver) UpdateBWPRefs(sCell, tCell *model.Cell, ue *model.UE) {
	//TODO: implement
}

// UpdateUESignalStrength updates UE signal strength
func (d *driver) UpdateUESignalStrength(imsi types.IMSI) {
	ue, ok := d.m.UEList[strconv.FormatUint(uint64(imsi), 10)]
	if !ok {
		log.Warn("Unable to find UE %d", imsi)
		return
	}

	// update RSRP from candidate serving cells
	err := d.updateUESignalStrengthCandServCells(&ue)
	if err != nil {
		log.Warnf("For UE %v: %v", ue, err)
		return
	}
}

// UpdateUESignalStrengthCandServCells updates UE signal strength for serving and candidate cells
func (d *driver) updateUESignalStrengthCandServCells(ue *model.UE) error {

	sCell := d.m.Cells[strconv.FormatUint(uint64(ue.Cell.NCGI), 10)]
	if !sCell.Cached {
		mpf := signal.RiceanFading(signal.GetRiceanK(&sCell))
		rsrp := signal.Strength(ue.Location, ue.Height, mpf, sCell)
		ue.Cell.Rsrp = rsrp
	}

	for _, ueCell := range ue.Cells {
		cell := d.m.Cells[strconv.FormatUint(uint64(ueCell.NCGI), 10)]
		if !sCell.Cached {
			mpf := signal.RiceanFading(signal.GetRiceanK(&cell))
			rsrp := signal.Strength(ue.Location, ue.Height, mpf, cell)
			if rsrp == math.Inf(-1) || math.IsNaN(rsrp) {
				continue
			}

			ueCell.Rsrp = rsrp
		}

	}
	d.m.UEList[strconv.FormatUint(uint64(ue.IMSI), 10)] = *ue
	// err := d.ueStore.UpdateCells(ctx, ue.IMSI, ue.Cells)
	// if err != nil {
	// 	log.Warn("Unable to update UE %d cells info", ue.IMSI)
	// }

	return nil
}

// SortUECells sorts ue cells
// func (d *driver) sortUECells(ueCells []*model.UECell, numAdjCells int) []*model.UECell {
// 	// bubble sort
// 	for i := 0; i < len(ueCells)-1; i++ {
// 		for j := 0; j < len(ueCells)-i-1; j++ {
// 			if ueCells[j].Rsrp < ueCells[j+1].Rsrp {
// 				ueCells[j], ueCells[j+1] = ueCells[j+1], ueCells[j]
// 			}
// 		}
// 	}
// 	if len(ueCells) >= numAdjCells {
// 		return ueCells[0:numAdjCells]
// 	}
// 	return ueCells
// }

// GetHoLogic returns the HO Logic ("local" or "mho")
func (d *driver) GetHoLogic() string {
	return d.hoLogic
}

// // DecrementRrcConnectedCount
// func (d *driver) DecrementRrcConnectedCount(ncgi types.NCGI) {
// 	d.lockCell(ncgi)
// 	defer d.cellLock[ncgi].Unlock()
// 	ncgiStr := strconv.FormatUint(uint64(ncgi), 10)
// 	cell := d.m.Cells[ncgiStr]
// 	if cell.RrcConnectedCount != 0 {
// 		cell.RrcConnectedCount--
// 	}

// 	d.m.Cells[ncgiStr] = cell
// }

// IncrementRrcConnectedCount
// func (d *driver) IncrementRrcConnectedCount(ncgi types.NCGI) {
// 	d.lockCell(ncgi)
// 	defer d.unlockCell(ncgi)
// 	ncgiStr := strconv.FormatUint(uint64(ncgi), 10)
// 	cell := d.m.Cells[ncgiStr]
// 	cell.RrcConnectedCount++
// 	d.m.Cells[ncgiStr] = cell
// }
