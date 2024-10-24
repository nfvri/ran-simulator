// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"fmt"
	"strconv"
	"time"

	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
	"github.com/nfvri/ran-simulator/pkg/handover"
	"github.com/nfvri/ran-simulator/pkg/mobility"
	"github.com/nfvri/ran-simulator/pkg/signal"
	"github.com/nfvri/ran-simulator/pkg/statistics"
	"github.com/nfvri/ran-simulator/pkg/store/routes"
	"github.com/nfvri/ran-simulator/pkg/utils"
	"github.com/sirupsen/logrus"

	cellapi "github.com/nfvri/ran-simulator/pkg/api/cells"
	metricsapi "github.com/nfvri/ran-simulator/pkg/api/metrics"
	modelapi "github.com/nfvri/ran-simulator/pkg/api/model"
	nodeapi "github.com/nfvri/ran-simulator/pkg/api/nodes"
	routeapi "github.com/nfvri/ran-simulator/pkg/api/routes"
	"github.com/nfvri/ran-simulator/pkg/api/trafficsim"
	ueapi "github.com/nfvri/ran-simulator/pkg/api/ues"
	"github.com/nfvri/ran-simulator/pkg/e2agent/agents"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/store/cells"
	"github.com/nfvri/ran-simulator/pkg/store/metrics"
	"github.com/nfvri/ran-simulator/pkg/store/nodes"
	redisLib "github.com/nfvri/ran-simulator/pkg/store/redis"
	uesstore "github.com/nfvri/ran-simulator/pkg/store/ues"
	e2smmho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"google.golang.org/grpc"

	"github.com/onosproject/onos-lib-go/pkg/northbound"
)

var log = logging.GetLogger()

// Config is a manager configuration
type Config struct {
	CAPath       string
	KeyPath      string
	CertPath     string
	GRPCPort     int
	ModelName    string
	MetricName   string
	HOLogic      string
	RedisEnabled bool
}

// NewManager creates a new manager
func NewManager(config *Config) (*Manager, error) {
	log.Info("Creating Manager")

	mgr := &Manager{
		config: *config,
		agents: nil,
		model:  &model.Model{},
	}

	return mgr, nil
}

// Manager is a manager for the E2T service
type Manager struct {
	modelapi.ManagementDelegate
	config         Config
	agents         *agents.E2Agents
	model          *model.Model
	server         *northbound.Server
	nodeStore      nodes.Store
	cellStore      cells.Store
	redisStore     redisLib.RedisStore
	ueStore        uesstore.Store
	routeStore     routes.Store
	metricsStore   metrics.Store
	mobilityDriver mobility.Driver
	finishHOsChan  chan bool
}

// Run starts the manager and the associated services
func (m *Manager) Run() {
	log.Info("Running Manager")
	if err := m.Start(); err != nil {
		log.Error("Unable to run Manager:", err)
	}
}

func (m *Manager) initMobilityDriver() {
	hoHandler := handover.NewA3HandoverHandler()
	ho := handover.NewA3Handover(hoHandler)
	hoCtrl := handover.NewHOController(handover.A3, ho)

	m.finishHOsChan = make(chan bool)

	m.mobilityDriver = mobility.NewMobilityDriver(
		m.model,
		m.config.HOLogic,
		hoCtrl,
		m.finishHOsChan,
	)
	ctx := context.Background()
	m.mobilityDriver.Start(ctx)
	for _, ue := range m.model.UEs {
		m.mobilityDriver.UpdateUESignalStrength(ue.IMSI)
	}
}

// Start starts the manager
func (m *Manager) Start() error {

	if m.config.RedisEnabled {
		redisHost := utils.GetEnv("REDIS_HOST", "localhost")
		redisPort := utils.GetEnv("REDIS_PORT", "6398")
		redisCellCache := utils.GetEnv("REDIS_CELL_CACHE_DB", "1")
		redisUECache := utils.GetEnv("REDIS_UE_CACHE_DB", "2")
		redisUsername := utils.GetEnv("REDIS_USERNAME", "")
		redisPass := utils.GetEnv("REDIS_PASSWORD", "")
		cellClient := redisLib.InitClient(redisHost, redisPort, redisCellCache, redisUsername, redisPass)
		ueClient := redisLib.InitClient(redisHost, redisPort, redisUECache, redisUsername, redisPass)
		m.redisStore = redisLib.RedisStore{
			CellDB: cellClient,
			UeDB:   ueClient,
		}
	}

	// Load the model data
	err := model.Load(m.model, m.config.ModelName)
	if err != nil {
		log.Error(err)
		return err
	}

	// m.initModelStores()
	m.initMetricStore()
	m.initMobilityDriver()

	// Start gRPC server
	err = m.startNorthboundServer()
	if err != nil {
		return err
	}

	// Start E2 agents
	// err = m.startE2Agents()
	// if err != nil {
	// 	return err
	// }

	return nil
}

// Close kills the channels and manager related objects
func (m *Manager) Close() {
	log.Info("Closing Manager")
	// m.stopE2Agents()
	m.stopNorthboundServer()
	m.mobilityDriver.Stop()
}

func (m *Manager) initMetricStore() {
	// Create store for tracking arbitrary metrics and attributes for nodes, cells and UEs
	m.metricsStore = metrics.NewMetricsStore()
}

func (m *Manager) computeCellAttributes() error {

	ueHeight := 1.5
	refSignalStrength := -107.0
	// change model's cells key from designated name to ncgi
	cellGroup := make(map[string]*model.Cell)
	for _, cell := range m.model.Cells {
		ncgi := strconv.FormatUint(uint64(cell.NCGI), 10)
		cellGroup[ncgi] = cell
	}
	m.model.Cells = cellGroup
	storeCells := signal.UpdateCells(m.model.Cells, &m.redisStore, ueHeight, refSignalStrength, m.model.DecorrelationDistance, m.model.SnapshotId)
	if storeCells {
		if err := m.redisStore.AddCellGroup(context.Background(), m.model.SnapshotId, m.model.Cells); err != nil {
			return fmt.Errorf("failed to store cells in cache: %v", err)
		}
		log.Infof("Updated CellGroup in Cache")
	}

	return nil
}

func (m *Manager) computeUEAttributes(ctx context.Context) {

	signal.PopulateUEs(m.model, &m.redisStore)

	numUEsByCell, prbMeasPerCell := bw.UtilizationInfoByCell(m.model.CellMeasurements)
	numUEsPerCQIByCell := bw.GetNumUEsPerCQIByCell(numUEsByCell)
	usedPRBsDLPerCQIByCell, usedPRBsULPerCQIByCell := bw.GetUsedPRBsPerCQIByCell(prbMeasPerCell, numUEsPerCQIByCell)

	for ncgi := range m.model.Cells {
		cell := m.model.Cells[ncgi]
		servedUEs := m.model.GetServedUEs(cell.NCGI)
		if len(servedUEs) == 0 {
			log.Warnf("number of ues for cell %v is 0", cell.NCGI)
			continue
		}

		statsPerCQI := map[int]bw.CQIStats{}
		for cqi, numUEs := range numUEsPerCQIByCell[uint64(cell.NCGI)] {
			if numUEs > 0 {
				statsPerCQI[cqi] = bw.CQIStats{
					NumUEs:     numUEs,
					UsedPRBsDL: usedPRBsDLPerCQIByCell[uint64(cell.NCGI)][cqi],
					UsedPRBsUL: usedPRBsULPerCQIByCell[uint64(cell.NCGI)][cqi],
				}
			}
		}

		availPRBsDL := prbMeasPerCell[uint64(cell.NCGI)][bw.AVAIL_PRBS_DL_METRIC]
		availPRBsUL := prbMeasPerCell[uint64(cell.NCGI)][bw.AVAIL_PRBS_UL_METRIC]

		m.setBWUtilization(ctx, cell, statsPerCQI, availPRBsDL, availPRBsUL)

		bw.InitBWPs(cell, statsPerCQI, availPRBsDL, availPRBsUL, servedUEs)
	}
}

func (m *Manager) setBWUtilization(ctx context.Context, cell *model.Cell, statsPerCQI map[int]bw.CQIStats, availPRBsDL, availPRBsUL int) {
	totalBWDL := bw.MHzToHz(float64(cell.Channel.BsChannelBwDL))
	totalBWUL := bw.MHzToHz(float64(cell.Channel.BsChannelBwUL))

	availBWDL := int(totalBWDL * bw.DEFAULT_MAX_BW_UTILIZATION)
	availBWUL := int(totalBWUL * bw.DEFAULT_MAX_BW_UTILIZATION)

	usedPRBsDL := 0
	usedPRBsUL := 0
	for _, cqiStats := range statsPerCQI {
		usedPRBsDL += cqiStats.UsedPRBsDL
		usedPRBsUL += cqiStats.UsedPRBsUL
	}

	bwUtilizationDL := float64(usedPRBsDL) / float64(availPRBsDL)
	bwUtilizationUL := float64(usedPRBsUL) / float64(availPRBsUL)

	m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.TOT_BW_USAGE_DL_METRIC, 100*bwUtilizationDL)
	m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.TOT_BW_USAGE_UL_METRIC, 100*bwUtilizationUL)
	m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.USED_BW_DL_METRIC, bwUtilizationDL*float64(availBWDL))
	m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.USED_BW_UL_METRIC, bwUtilizationUL*float64(availBWUL))
}

func (m *Manager) computeCellStatistics(ctx context.Context) {

	totalactiveUEs := 0
	totalPrbsTotalDl := 0
	totalPrbsTotalUl := 0

	prbsUsedDLPerCQI := map[int]int{}
	prbsUsedULPerCQI := map[int]int{}

	for _, cell := range m.model.Cells {
		servedUEs := m.model.GetServedUEs(cell.NCGI)
		prbsUsedDl := 0
		bwUsedDl := 0
		prbsUsedUl := 0
		bwUsedUl := 0
		activeUEs := 0

		if len(cell.Bwps) == 0 {
			log.Warnf("cell %v Bwps: %v", cell.NCGI, cell.Bwps)
		}

		for _, ue := range servedUEs {

			if _, ok := prbsUsedDLPerCQI[ue.FiveQi]; !ok {
				prbsUsedDLPerCQI[ue.FiveQi] = 0
			}
			if _, ok := prbsUsedULPerCQI[ue.FiveQi]; !ok {
				prbsUsedULPerCQI[ue.FiveQi] = 0
			}

			if ue.RrcState == e2smmho.Rrcstatus_RRCSTATUS_CONNECTED {
				activeUEs++
			}

			for _, bwp := range ue.Cell.BwpRefs {
				if bwp.Downlink {
					prbsUsedDl += bwp.NumberOfRBs
					bwUsedDl += 12 * bwp.NumberOfRBs * bwp.Scs
					prbsUsedDLPerCQI[ue.FiveQi] += bwp.NumberOfRBs
				} else {
					prbsUsedUl += bwp.NumberOfRBs
					bwUsedUl += 12 * bwp.NumberOfRBs * bwp.Scs
					prbsUsedULPerCQI[ue.FiveQi] += bwp.NumberOfRBs
				}
			}
		}
		totalactiveUEs += activeUEs
		totalPrbsTotalDl += prbsUsedDl
		totalPrbsTotalUl += prbsUsedUl

		iBwUtilizationDL, _ := m.metricsStore.Get(ctx, uint64(cell.NCGI), bw.TOT_BW_USAGE_DL_METRIC)
		prevBwUtilizationDL := iBwUtilizationDL.(float64)
		iBwUtilizationUL, _ := m.metricsStore.Get(ctx, uint64(cell.NCGI), bw.TOT_BW_USAGE_UL_METRIC)
		prevBwUtilizationUL := iBwUtilizationUL.(float64)

		iUsedBWDL, _ := m.metricsStore.Get(ctx, uint64(cell.NCGI), bw.USED_BW_DL_METRIC)
		prevUsedBWDL := iUsedBWDL.(float64)
		iUsedBWUL, _ := m.metricsStore.Get(ctx, uint64(cell.NCGI), bw.USED_BW_UL_METRIC)
		prevUsedBWUL := iUsedBWUL.(float64)

		logrus.Info("======================================")
		logrus.Infof("ncgi: %v", cell.NCGI)
		logrus.Infof("prevBwUtilizationDL: %v", prevBwUtilizationDL)
		logrus.Infof("prevBwUtilizationUL: %v", prevBwUtilizationUL)
		logrus.Infof("prevUsedBWDL: %.f", prevUsedBWDL)
		logrus.Infof("prevUsedBWUL: %.f", prevUsedBWUL)
		logrus.Info("----------------------------------------")

		m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.USED_PRBS_DL_METRIC, prbsUsedDl)
		m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.USED_PRBS_UL_METRIC, prbsUsedUl)
		for cqi, prbsDl := range prbsUsedDLPerCQI {
			m.metricsStore.Set(ctx, uint64(cell.NCGI), fmt.Sprintf(bw.USED_PRBS_DL_METRIC+".%d", cqi), prbsDl)
		}
		for cqi, prbsUl := range prbsUsedULPerCQI {
			m.metricsStore.Set(ctx, uint64(cell.NCGI), fmt.Sprintf(bw.USED_PRBS_UL_METRIC+".%d", cqi), prbsUl)
		}
		m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.ACTIVE_UES_DL_METRIC, activeUEs)
		m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.ACTIVE_UES_UL_METRIC, activeUEs)
		m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.UE_THP_DL_METRIC, statistics.UEThp(prbsUsedDl, len(servedUEs)))
		m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.UE_THP_UL_METRIC, statistics.UEThp(prbsUsedUl, len(servedUEs)))

		totalBWDL := bw.MHzToHz(float64(cell.Channel.BsChannelBwDL))
		totalBWUL := bw.MHzToHz(float64(cell.Channel.BsChannelBwUL))

		availBWDL := int(totalBWDL * bw.DEFAULT_MAX_BW_UTILIZATION)
		availBWUL := int(totalBWUL * bw.DEFAULT_MAX_BW_UTILIZATION)

		bwUtilizationDL := float64(bwUsedDl) / float64(availBWDL)
		bwUtilizationUL := float64(bwUsedUl) / float64(availBWUL)
		m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.TOT_BW_USAGE_DL_METRIC, 100*bwUtilizationDL)
		m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.TOT_BW_USAGE_DL_METRIC, 100*bwUtilizationUL)
		m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.USED_BW_DL_METRIC, bwUsedDl)
		m.metricsStore.Set(ctx, uint64(cell.NCGI), bw.USED_BW_UL_METRIC, bwUsedUl)

		logrus.Infof("BwUtilizationDL: %v", 100*bwUtilizationDL)
		logrus.Infof("BwUtilizationUL: %v", 100*bwUtilizationUL)
		logrus.Infof("usedBWDL: %.f", float64(bwUsedDl))
		logrus.Infof("usedBWUL: %.f", float64(bwUsedUl))
		logrus.Infof("availBWDL: %.f", float64(availBWDL))
		logrus.Infof("availBWUL: %.f", float64(availBWUL))
		logrus.Info("======================================\n")
	}

	m.metricsStore.Set(ctx, uint64(1), "SUBNET_RRU.PrbTotDl", totalPrbsTotalDl)
	m.metricsStore.Set(ctx, uint64(1), "SUBNET_RRU.PrbTotUl", totalPrbsTotalUl)
	m.metricsStore.Set(ctx, uint64(1), "SUBNET_AVG_DRB.UEThpDl", statistics.UEThp(totalPrbsTotalDl, totalactiveUEs))
	m.metricsStore.Set(ctx, uint64(1), "SUBNET_AVG_DRB.UEThpUl", statistics.UEThp(totalPrbsTotalUl, totalactiveUEs))
	m.metricsStore.Set(ctx, uint64(1), "SUBNET_DRB.MeanActiveUeDl", totalactiveUEs)
}

// startSouthboundServer starts the northbound gRPC server
func (m *Manager) startNorthboundServer() error {
	m.server = northbound.NewServer(northbound.NewServerCfg(
		m.config.CAPath,
		m.config.KeyPath,
		m.config.CertPath,
		int16(m.config.GRPCPort),
		true,
		northbound.SecurityConfig{}))

	m.server.AddService(logging.Service{})
	m.server.AddService(nodeapi.NewService(m.nodeStore, m.model.PlmnID))
	m.server.AddService(cellapi.NewService(m.cellStore))
	m.server.AddService(trafficsim.NewService(m.model, m.cellStore, m.ueStore))
	m.server.AddService(metricsapi.NewService(m.metricsStore))
	m.server.AddService(ueapi.NewService(m.ueStore))
	m.server.AddService(routeapi.NewService(m.routeStore))
	m.server.AddService(modelapi.NewService(m))

	maxMsgSize := 32 * 1024 * 1024 // 32MB
	grpcOpts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	}

	doneCh := make(chan error)
	go func() {
		err := m.server.Serve(
			func(started string) {
				log.Info("Started NBI on ", started)
				close(doneCh)
			},
			grpcOpts...,
		)
		if err != nil {
			doneCh <- err
		}
	}()
	return <-doneCh
}

func (m *Manager) StartE2Agents() error {
	// Create the E2 agents for all simulated nodes and specified controllers
	// var err error
	// m.agents, err = agents.NewE2Agents(m.model, m.nodeStore, m.ueStore, m.cellStore, m.metricsStore, m.mobilityDriver.GetHoCtrl().GetOutputChan(), m.mobilityDriver)
	// if err != nil {
	// 	log.Error(err)
	// 	return err
	// }
	// // Start the E2 agents
	// err = m.agents.Start()
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (m *Manager) StopE2Agents() {
	_ = m.agents.Stop()
}

func (m *Manager) stopNorthboundServer() {
	m.server.Stop()
}

// PauseAndClear pauses simulation and clears the model
func (m *Manager) PauseAndClear(ctx context.Context) {
	log.Info("Pausing RAN simulator...")
	m.metricsStore.Clear(ctx)
}

// LoadModel loads the new model into the simulator
func (m *Manager) LoadModel(ctx context.Context, data []byte) error {
	m.model = &model.Model{}
	if err := model.LoadConfigFromBytes(m.model, data); err != nil {
		return err
	}

	m.LoadMetrics(ctx)
	return nil
}

// LoadMetrics loads new metrics into the simulator
func (m *Manager) LoadMetrics(ctx context.Context) error {
	for _, metric := range m.model.CellMeasurements {
		m.metricsStore.Set(ctx, metric.EntityID, metric.Key, metric.Value)
	}
	return nil
}

// Resume resume the simulation
func (m *Manager) Resume(ctx context.Context) error {
	log.Info("Resuming RAN simulator...")
	// _ = m.StartE2Agents()

	if err := m.computeCellAttributes(); err != nil {
		return err
	}
	log.Info("\n====[IN MANAGER1]====\n")
	for _, cell := range m.model.Cells {
		if len(cell.Bwps) > 0 {
			log.Infof("NCGI: %v len(bwps): %v", cell.NCGI, len(cell.Bwps))
		}
	}

	m.computeUEAttributes(ctx)
	m.initMobilityDriver()
	m.performHandovers()
	m.computeCellStatistics(ctx)
	go func() {
		time.Sleep(1 * time.Millisecond)
		log.Info("Restarting NBI...")
		m.stopNorthboundServer()
		_ = m.startNorthboundServer()
	}()

	return nil
}

func (m *Manager) performHandovers() {

	hoUes := map[string]model.UE{}
	for imsi := range m.model.UEs {
		ue := *m.model.UEs[imsi]
		hoUes[imsi] = ue
	}

	for imsi := range hoUes {
		ue := hoUes[imsi]
		m.mobilityDriver.GetHoCtrl().GetInputChan() <- ue
	}

	defer close(m.finishHOsChan)
	for range m.finishHOsChan {
		log.Info("HOs completed")
		return
	}
}
