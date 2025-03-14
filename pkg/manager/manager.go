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

const NUM_SUBFRAMES = 10

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
	logrus.Info("Creating Manager")

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
	patchedUEs     []model.UE
	patchedCells   []model.Cell
}

// Run starts the manager and the associated services
func (m *Manager) Run() {
	logrus.Info("Running Manager")
	if err := m.Start(); err != nil {
		logrus.Error("Unable to run Manager:", err)
	}
}

func (m *Manager) initMobilityDriver() {
	hoHandler := handover.NewA3HandoverHandler(m.model)
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
		logrus.Error(err)
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
	logrus.Info("Closing Manager")
	// m.stopE2Agents()
	m.stopNorthboundServer()
	m.mobilityDriver.Stop()
}

func (m *Manager) createModelStores(ctx context.Context) {
	// Create the node registry primed with the model nodes
	m.nodeStore = nodes.NewNodeRegistry(ctx, m.model.Nodes)

	// Create the cell registry primed with the model cells
	m.cellStore = cells.NewCellRegistry(ctx, m.model.Cells, m.nodeStore)

	// Create the ue registry primed with the model ues
	m.ueStore = uesstore.NewUERegistry(ctx, m.model, m.cellStore, m.model.InitialRrcState)

}

func (m *Manager) createPatchedStores() {

	m.patchedCells = []model.Cell{}
	for ncgi := range m.model.Cells {
		cell := *m.model.Cells[ncgi]
		cellBwps := map[uint64]*model.Bwp{}
		for index := range cell.Bwps {
			bwp := *cell.Bwps[index]
			cellBwps[index] = &bwp
		}
		cell.Bwps = cellBwps
		m.patchedCells = append(m.patchedCells, cell)
	}

	m.patchedUEs = []model.UE{}
	for imsi := range m.model.UEs {
		ue := *m.model.UEs[imsi]
		ueCell := *ue.ServingCells[0]
		ue.ServingCells[0] = &ueCell

		ueCells := []*model.UECell{}
		for index := range ue.ServingCells {
			ueCellcp := *ue.ServingCells[index]
			ueCells = append(ueCells, &ueCellcp)
		}
		ue.ServingCells = ueCells

		m.patchedUEs = append(m.patchedUEs, ue)
	}
}

func (m *Manager) initMetricStore() {
	// Create store for tracking arbitrary metrics and attributes for nodes, cells and UEs
	m.metricsStore = metrics.NewMetricsStore()
}

func (m *Manager) computeCellAttributes() error {

	ueHeight := 1.5
	refSignalStrength := -87.0
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
		logrus.Infof("Updated CellGroup in Cache")
	}

	return nil
}

func (m *Manager) initCellMetrics(ctx context.Context) {

	signal.PopulateUEs(m.model, &m.redisStore)

	numUEsByCell, prbMeasPerCell := bw.UtilizationInfoByCell(m.model.CellMeasurements)
	numUEsPerCQIByCell := bw.GetNumUEsPerCQIByCell(numUEsByCell)
	usedPRBsDLPerCQIByCell, usedPRBsULPerCQIByCell := bw.GetUsedPRBsPerCQIByCell(prbMeasPerCell, numUEsPerCQIByCell)
	usedPRBsDLPerCQIByCell = bw.CheckBWOverflow(usedPRBsDLPerCQIByCell, prbMeasPerCell, bw.AVAIL_PRBS_DL_METRIC)
	usedPRBsULPerCQIByCell = bw.CheckBWOverflow(usedPRBsULPerCQIByCell, prbMeasPerCell, bw.AVAIL_PRBS_UL_METRIC)

	for ncgi := range m.model.Cells {

		cell := m.model.Cells[ncgi]
		servedUEs := m.model.GetServedUEs(cell.NCGI)

		usedPRBsDLPerCQI := usedPRBsDLPerCQIByCell[uint64(cell.NCGI)]
		usedPRBsULPerCQI := usedPRBsULPerCQIByCell[uint64(cell.NCGI)]
		numUEs := numUEsPerCQIByCell[uint64(cell.NCGI)]
		availPRBsDL := prbMeasPerCell[uint64(cell.NCGI)][bw.AVAIL_PRBS_DL_METRIC]
		availPRBsUL := prbMeasPerCell[uint64(cell.NCGI)][bw.AVAIL_PRBS_UL_METRIC]

		sumUsedPRBsDL := 0
		sumUsedPRBsUL := 0
		for _, usedPRBs := range usedPRBsDLPerCQI {
			sumUsedPRBsDL += usedPRBs
		}
		for _, usedPRBs := range usedPRBsULPerCQI {
			sumUsedPRBsUL += usedPRBs
		}

		prbUtilizationDL := 0.0
		if sumUsedPRBsDL != 0 || availPRBsDL != 0 {
			prbUtilizationDL = float64(sumUsedPRBsDL) / float64(availPRBsDL)
		}
		prbUtilizationUL := 0.0
		if sumUsedPRBsUL != 0 || availPRBsUL != 0 {
			prbUtilizationUL = float64(sumUsedPRBsUL) / float64(availPRBsUL)
		}

		prbUtilStats := map[string]any{
			bw.PRBS_UTIL_DL_METRIC:  utils.RoundToDecimal(prbUtilizationDL, 4),
			bw.PRBS_UTIL_UL_METRIC:  utils.RoundToDecimal(prbUtilizationUL, 4),
			bw.USED_PRBS_DL_METRIC:  sumUsedPRBsDL,
			bw.USED_PRBS_UL_METRIC:  sumUsedPRBsUL,
			bw.AVAIL_PRBS_DL_METRIC: availPRBsDL,
			bw.AVAIL_PRBS_UL_METRIC: availPRBsUL,
		}

		m.storeStats(ctx, uint64(cell.NCGI), prbUtilStats)
		bw.AllocatePRBs(cell, numUEs, usedPRBsDLPerCQI, usedPRBsULPerCQI, availPRBsDL, availPRBsUL, servedUEs)

		if len(cell.Bwps) == 0 && sumUsedPRBsDL+sumUsedPRBsUL != 0 {
			logrus.Errorf("failed to initialize BWPs for cell: %v", cell.NCGI)
		}
	}
}

func (m *Manager) computeCellStatistics(ctx context.Context) {

	logrus.Info(`
	------------------------------------
	CALCULATING STATISTICS
	------------------------------------
	`)

	_, prbMeasPerCell := bw.UtilizationInfoByCell(m.model.CellMeasurements)

	totalActiveUEs := 0
	totalPrbsDl := 0
	totalPrbsUl := 0

	for _, cell := range m.model.Cells {
		prbsUsedDLPerCQI := map[int]int{}
		prbsUsedULPerCQI := map[int]int{}

		servedUEs := m.model.GetServedUEs(cell.NCGI)
		prbsUsedDl := 0
		prbsUsedUl := 0
		activeUEs := 0

		if len(cell.Bwps) == 0 {
			logrus.Warnf("cell %v Bwps: %v", cell.NCGI, cell.Bwps)
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

			for sCellIndex := range ue.ServingCells {
				sCell := ue.ServingCells[sCellIndex]
				if sCell.NCGI == cell.NCGI {

					for _, bwp := range sCell.BwpRefs {
						framePRBs := int(float64(bwp.NumberOfRBs) / float64((bwp.Scs / 15)))
						if bwp.Downlink {
							prbsUsedDl += framePRBs
							prbsUsedDLPerCQI[ue.FiveQi] += framePRBs
						} else {
							prbsUsedUl += framePRBs
							prbsUsedULPerCQI[ue.FiveQi] += framePRBs
						}
					}

					break
				}
			}
		}

		prbsAvailDL := prbMeasPerCell[uint64(cell.NCGI)][bw.AVAIL_PRBS_DL_METRIC]
		prbsAvailUL := prbMeasPerCell[uint64(cell.NCGI)][bw.AVAIL_PRBS_UL_METRIC]
		prbUtilDL := float64(prbsUsedDl) / float64(prbsAvailDL)
		prbUtilUL := float64(prbsUsedUl) / float64(prbsAvailUL)

		totalActiveUEs += activeUEs
		totalPrbsDl += prbsUsedDl
		totalPrbsUl += prbsUsedUl

		arfcn := utils.If(cell.ArfcnDL > 0, cell.ArfcnDL, cell.ArfcnUL)
		direction := utils.If(cell.ArfcnDL > 0, bw.DL, bw.UL)
		operatingBand, _ := bw.GetBandNR(arfcn, direction)

		activeUEsUL := 0
		activeUEsDL := 0

		switch operatingBand.DuplexingMode {
		case bw.FDD, bw.TDD:
			activeUEsUL = activeUEs
			activeUEsDL = activeUEs
		case bw.SUL:
			activeUEsUL = activeUEs
		case bw.SDL:
			activeUEsDL = activeUEs
		}

		// logrus.Infof(`
		// ====================================================================
		// ncgi: %v
		// operatingBand: %v
		// duplex mode: %v
		// ====================================================================
		// 	`,
		// 	cell.NCGI,
		// 	operatingBand.Name,
		// 	operatingBand.DuplexingMode)
		// m.logActiveUEs(ctx, cell)
		// m.logPRBUtilization(ctx, cell)

		// TODO:
		// statistics.CalculateThroughputMbps()

		cellStats := map[string]any{
			bw.ACTIVE_UES_DL_METRIC: activeUEsDL,
			bw.ACTIVE_UES_UL_METRIC: activeUEsUL,
			bw.UE_THP_DL_METRIC:     statistics.UEThp(prbsUsedDl, len(servedUEs)),
			bw.UE_THP_UL_METRIC:     statistics.UEThp(prbsUsedUl, len(servedUEs)),
			bw.PRBS_UTIL_DL_METRIC:  utils.RoundToDecimal(prbUtilDL, 4),
			bw.PRBS_UTIL_UL_METRIC:  utils.RoundToDecimal(prbUtilUL, 4),
			bw.USED_PRBS_DL_METRIC:  prbsUsedDl,
			bw.USED_PRBS_UL_METRIC:  prbsUsedUl,
			bw.AVAIL_PRBS_DL_METRIC: prbsAvailDL,
			bw.AVAIL_PRBS_UL_METRIC: prbsAvailUL,
		}

		for cqi, prbsDl := range prbsUsedDLPerCQI {
			cellStats[fmt.Sprintf(bw.USED_PRBS_DL_METRIC+".%d", cqi)] = prbsDl
		}
		for cqi, prbsUl := range prbsUsedULPerCQI {
			cellStats[fmt.Sprintf(bw.USED_PRBS_UL_METRIC+".%d", cqi)] = prbsUl
		}

		m.storeStats(ctx, uint64(cell.NCGI), cellStats)

		// m.logActiveUEs(ctx, cell)
		// m.logPRBUtilization(ctx, cell)
	}

	subnetStats := map[string]any{
		"SUBNET_RRU.PrbTotDl":       totalPrbsDl,
		"SUBNET_RRU.PrbTotUl":       totalPrbsUl,
		"SUBNET_AVG_DRB.UEThpDl":    statistics.UEThp(totalPrbsDl, totalActiveUEs),
		"SUBNET_AVG_DRB.UEThpUl":    statistics.UEThp(totalPrbsUl, totalActiveUEs),
		"SUBNET_DRB.MeanActiveUeDl": totalActiveUEs,
	}

	m.storeStats(ctx, uint64(1), subnetStats)

}

func (m *Manager) logActiveUEs(ctx context.Context, cell *model.Cell) {

	iActiveUEsDL, _ := m.metricsStore.Get(ctx, uint64(cell.NCGI), bw.ACTIVE_UES_DL_METRIC)
	activeUEsDL := 0
	if iActiveUEsDL != nil {
		activeUEsDL = iActiveUEsDL.(int)
	}

	iActiveUEsUL, _ := m.metricsStore.Get(ctx, uint64(cell.NCGI), bw.ACTIVE_UES_UL_METRIC)
	activeUEsUL := 0
	if iActiveUEsUL != nil {
		activeUEsUL = iActiveUEsUL.(int)
	}

	logrus.Info("--------------------------------------------------------------------")
	logrus.Infof("ncgi: %v | Active UEs DL: %v", cell.NCGI, activeUEsDL)
	logrus.Infof("ncgi: %v | Active UEs UL: %v", cell.NCGI, activeUEsUL)
}

func (m *Manager) logPRBUtilization(ctx context.Context, cell *model.Cell) {

	iPRBUtilizationDL, _ := m.metricsStore.Get(ctx, uint64(cell.NCGI), bw.PRBS_UTIL_DL_METRIC)
	prbUtilizationDL := iPRBUtilizationDL.(float64)
	iPRBUtilizationUL, _ := m.metricsStore.Get(ctx, uint64(cell.NCGI), bw.PRBS_UTIL_UL_METRIC)
	prbUtilizationUL := iPRBUtilizationUL.(float64)

	iUsedPRBsDL, _ := m.metricsStore.Get(ctx, uint64(cell.NCGI), bw.USED_PRBS_DL_METRIC)
	usedPRBsDL := iUsedPRBsDL.(int)
	iUsedPRBsUL, _ := m.metricsStore.Get(ctx, uint64(cell.NCGI), bw.USED_PRBS_UL_METRIC)
	usedPRBsUL := iUsedPRBsUL.(int)

	iAvailPRBsDL, _ := m.metricsStore.Get(ctx, uint64(cell.NCGI), bw.AVAIL_PRBS_DL_METRIC)
	availPRBsDL := iAvailPRBsDL.(int)
	iAvailPRBsUL, _ := m.metricsStore.Get(ctx, uint64(cell.NCGI), bw.AVAIL_PRBS_UL_METRIC)
	availPRBsUL := iAvailPRBsUL.(int)

	logrus.Infof("ncgi: %v | PRB Utilization DL: %v", cell.NCGI, utils.RoundToDecimal(prbUtilizationDL, 4))
	logrus.Infof("ncgi: %v | PRB Utilization UL: %v", cell.NCGI, utils.RoundToDecimal(prbUtilizationUL, 4))
	logrus.Infof("ncgi: %v | PRBs Used DL: %d", cell.NCGI, usedPRBsDL)
	logrus.Infof("ncgi: %v | PRBs Used UL: %d", cell.NCGI, usedPRBsUL)
	logrus.Infof("ncgi: %v | PRBs Available DL: %d", cell.NCGI, availPRBsDL)
	logrus.Infof("ncgi: %v | PRBs Available UL: %d", cell.NCGI, availPRBsUL)
	logrus.Info("--------------------------------------------------------------------")
}

func (m *Manager) storeStats(ctx context.Context, entityID uint64, stats map[string]any) {
	for k, v := range stats {
		m.metricsStore.Set(ctx, entityID, k, v)
	}
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
	m.server.AddService(cellapi.NewService(m.cellStore, m.patchedCells))
	m.server.AddService(trafficsim.NewService(m.model, m.cellStore, m.ueStore))
	m.server.AddService(metricsapi.NewService(m.metricsStore))
	m.server.AddService(ueapi.NewService(m.ueStore, m.patchedUEs))
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
				logrus.Info("Started NBI on ", started)
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
	// 	logrus.Error(err)
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
	logrus.Info("Pausing RAN simulator...")
	m.metricsStore.Clear(ctx)
}

// LoadModel loads the new model into the simulator
func (m *Manager) LoadModel(ctx context.Context, data []byte) error {
	m.model = &model.Model{}
	if err := model.LoadConfigFromBytes(m.model, data); err != nil {
		return err
	}
	now := time.Now()

	m.model.CreationTimestamp = now.Format("2006-01-02 15:04:05") // Example format: "YYYY-MM-DD HH:MM:SS"

	m.LoadMetrics(ctx)
	return nil
}

func (m *Manager) GetModel(ctx context.Context) (*model.Model, error) {

	if m.model == nil || m.model.SnapshotId == "" {
		return nil, fmt.Errorf("no model is loaded in ransim")
	}

	return m.model, nil
}

// LoadMetrics loads new metrics into the simulator
func (m *Manager) LoadMetrics(ctx context.Context) error {
	for _, metric := range m.model.CellMeasurements {

		iValue, err := strconv.Atoi(metric.Value)
		if err == nil {
			m.metricsStore.Set(ctx, metric.EntityID, metric.Key, iValue)
			continue
		}

		fValue, err := strconv.ParseFloat(metric.Value, 64)
		if err == nil {
			m.metricsStore.Set(ctx, metric.EntityID, metric.Key, fValue)
			continue
		}

		m.metricsStore.Set(ctx, metric.EntityID, metric.Key, metric.Value)

	}
	return nil
}

// Resume resume the simulation
func (m *Manager) Resume(ctx context.Context) error {
	logrus.Info("Resuming RAN simulator...")
	// _ = m.StartE2Agents()

	if err := m.computeCellAttributes(); err != nil {
		return err
	}
	for _, cell := range m.model.Cells {
		if len(cell.Bwps) > 0 {
			logrus.Infof("NCGI: %v len(bwps): %v", cell.NCGI, len(cell.Bwps))
		}
	}

	m.initCellMetrics(ctx)
	m.initMobilityDriver()
	m.createPatchedStores()

	m.performHandovers()
	m.computeCellStatistics(ctx)
	m.createModelStores(ctx)
	go func() {
		time.Sleep(1 * time.Millisecond)
		logrus.Info("Restarting NBI...")
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
		logrus.Info("HOs completed")
		return
	}
}
