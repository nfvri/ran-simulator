// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"time"

	"github.com/nfvri/ran-simulator/pkg/mobility"
	"github.com/nfvri/ran-simulator/pkg/store/routes"
	"github.com/nfvri/ran-simulator/pkg/utils"

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
	"github.com/nfvri/ran-simulator/pkg/store/ues"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/redis/go-redis/v9"
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
	ueStore        ues.Store
	routeStore     routes.Store
	metricsStore   metrics.Store
	mobilityDriver mobility.Driver
	rdbClient      *redis.Client
}

// Run starts the manager and the associated services
func (m *Manager) Run() {
	log.Info("Running Manager")
	if err := m.Start(); err != nil {
		log.Error("Unable to run Manager:", err)
	}
}

func (m *Manager) initmobilityDriver() {
	m.mobilityDriver = mobility.NewMobilityDriver(m.cellStore, m.routeStore, m.ueStore, m.model.APIKey, m.config.HOLogic, m.model.UECountPerCell, m.model.RrcStateChangesDisabled, m.model.WayPointRoute)
	// m.mobilityDriver.GenerateRoutes(context.Background(), 720000, 1080000, 20000, m.model.RouteEndPoints, m.model.DirectRoute)
	// m.mobilityDriver.Start(context.Background())
	cellList, _ := m.cellStore.List(context.Background())
	for _, cell := range cellList {
		cellShadowMap, err := redisLib.GetShadowMapByNCGI(m.rdbClient, uint64(cell.NCGI))
		if err != nil {
			initializeCellShadowMap(cell, m)
		} else {
			cell.GridPoints = cellShadowMap.GridPoints
			cell.ShadowingMap = cellShadowMap.ShadowingMap
		}
	}
	ueList := m.ueStore.ListAllUEs(context.Background())
	for _, ue := range ueList {
		m.mobilityDriver.UpdateUESignalStrength(context.Background(), ue.IMSI)
	}

}

func initializeCellShadowMap(cell *model.Cell, m *Manager) {
	log.Warnf("failed to retrieve shadowmap for cell: %d", cell.NCGI)
	m.mobilityDriver.InitShadowMap(cell, m.model.DecorrelationDistance)
	err := redisLib.AddShadowMap(m.rdbClient, uint64(cell.NCGI),
		&model.ShadowMap{
			ShadowingMap: cell.ShadowingMap,
			GridPoints:   cell.GridPoints,
		})
	if err != nil {
		log.Errorf("failed to store shadowmap for cell: %d", cell.NCGI)
	}
}

// Start starts the manager
func (m *Manager) Start() error {

	if m.config.RedisEnabled {
		redisHost := utils.GetEnv("REDIS_HOST", "localhost")
		redisPort := utils.GetEnv("REDIS_PORT", "6379")
		rdbClient := redisLib.InitClient(redisHost, redisPort)
		m.rdbClient = rdbClient
	}

	// Load the model data
	err := model.Load(m.model, m.config.ModelName)
	if err != nil {
		log.Error(err)
		return err
	}

	m.initModelStores()
	m.initMetricStore()

	// Start gRPC server
	err = m.startNorthboundServer()
	if err != nil {
		return err
	}

	m.initmobilityDriver()

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

func (m *Manager) initModelStores() {
	// Create the node registry primed with the pre-loaded nodes
	m.nodeStore = nodes.NewNodeRegistry(m.model.Nodes)

	// Create the cell registry primed with the pre-loaded cells
	m.cellStore = cells.NewCellRegistry(m.model.Cells, m.nodeStore)

	// Create the UE registry primed with the specified number of UEs
	m.ueStore = ues.NewUERegistry(*m.model, m.cellStore, m.model.InitialRrcState)

	// Create an empty route registry
	// m.routeStore = routes.NewRouteRegistry()
}

func (m *Manager) initMetricStore() {
	// Create store for tracking arbitrary metrics and attributes for nodes, cells and UEs
	m.metricsStore = metrics.NewMetricsStore()
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

	doneCh := make(chan error)
	go func() {
		err := m.server.Serve(func(started string) {
			log.Info("Started NBI on ", started)
			close(doneCh)
		})
		if err != nil {
			doneCh <- err
		}
	}()
	return <-doneCh
}

func (m *Manager) StartE2Agents() error {
	// Create the E2 agents for all simulated nodes and specified controllers
	var err error
	m.agents, err = agents.NewE2Agents(m.model, m.nodeStore, m.ueStore, m.cellStore, m.metricsStore, m.mobilityDriver.GetHoCtrl().GetOutputChan(), m.mobilityDriver)
	if err != nil {
		log.Error(err)
		return err
	}
	// Start the E2 agents
	err = m.agents.Start()
	if err != nil {
		return err
	}

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
	// m.stopE2Agents()
	m.nodeStore.Clear(ctx)
	m.cellStore.Clear(ctx)
	m.metricsStore.Clear(ctx)
	m.mobilityDriver.Stop()
}

// LoadModel loads the new model into the simulator
func (m *Manager) LoadModel(ctx context.Context, data []byte) error {
	m.model = &model.Model{}
	if err := model.LoadConfigFromBytes(m.model, data); err != nil {
		return err
	}
	m.initModelStores()
	return nil
}

// LoadMetrics loads new metrics into the simulator
func (m *Manager) LoadMetrics(ctx context.Context, name string, data []byte) error {
	// TODO: Deprecated; remove this
	return nil
}

// Resume resume the simulation
func (m *Manager) Resume(ctx context.Context) {
	log.Info("Resuming RAN simulator...")
	go func() {
		time.Sleep(1 * time.Second)
		log.Info("Restarting NBI...")
		m.stopNorthboundServer()
		_ = m.startNorthboundServer()
	}()
	// _ = m.startE2Agents()
	m.initmobilityDriver()
}
