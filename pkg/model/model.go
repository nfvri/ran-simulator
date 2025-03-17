// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/nfvri/onos-api/go/onos/ransim/metrics"
	"github.com/nfvri/onos-api/go/onos/ransim/types"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Model simulation model
type Model struct {
	MapLayout               MapLayout               `mapstructure:"layout" yaml:"layout"`
	RouteEndPoints          []RouteEndPoint         `mapstructure:"routeEndPoints" yaml:"routeEndPoints"`
	WayPointRoute           bool                    `mapstructure:"wayPointRoute" yaml:"wayPointRoute"`
	DirectRoute             bool                    `mapstructure:"directRoute" yaml:"directRoute"`
	Nodes                   map[string]Node         `mapstructure:"nodes" yaml:"nodes"`
	Cells                   map[string]*Cell        `mapstructure:"cells" yaml:"cells"`
	Controllers             map[string]Controller   `mapstructure:"controllers" yaml:"controllers"`
	ServiceModels           map[string]ServiceModel `mapstructure:"servicemodels" yaml:"servicemodels"`
	RrcStateChangesDisabled bool                    `mapstructure:"RrcStateChangesDisabled" yaml:"RrcStateChangesDisabled"`
	InitialRrcState         string                  `mapstructure:"initialRrcState" yaml:"initialRrcState"`
	UECount                 uint                    `mapstructure:"ueCount" yaml:"ueCount"`
	UECountPerCell          uint                    `mapstructure:"ueCountPerCell" yaml:"ueCountPerCell"`
	UEs                     map[string]*UE          `mapstructure:"ues" yaml:"ues"`
	Plmn                    string                  `mapstructure:"plmnID" yaml:"plmnID"`
	PlmnID                  types.PlmnID            `mapstructure:"plmnNumber" yaml:"plmnNumber"` // overridden and derived post-load from "Plmn" field
	APIKey                  string                  `mapstructure:"apiKey" yaml:"apiKey"`         // Google Maps API key (optional)
	Guami                   Guami                   `mapstructure:"guami" yaml:"guami"`
	DecorrelationDistance   float64                 `mapstructure:"decorrelationdistance" yaml:"decorrelationdistance"`
	SnapshotId              string                  `mapstructure:"snapshotID" yaml:"snapshotID" ` //used to retrieve snapshot Cell Group and UE Group
	CellMeasurements        []*metrics.Metric       `mapstructure:"cellMeasurements" yaml:"cellMeasurements"`
	CreationTimestamp       string                  `mapstructure:"creationTimestamp" yaml:"creationTimestamp"`
	ServiceMappings
}

func (m *Model) UpdateServiceMappings(ueIMSI types.IMSI, sourceCellNcgis, targetCellINcgis []types.NCGI) {

	logrus.Infof("ue: %v | [UPDATE-SM] sourceCellNcgis: %v, targetCellINcgis: %v", ueIMSI, sourceCellNcgis, targetCellINcgis)

	// delete ue from sourceCells & sourceCells from ue
	ue := m.UEs[strconv.FormatUint(uint64(ueIMSI), 10)]
	for _, scNcgi := range sourceCellNcgis {

		for i, imsi := range m.CellToUEs[scNcgi] {
			if imsi == ueIMSI {
				m.CellToUEs[scNcgi] = append(m.CellToUEs[scNcgi][:i], m.CellToUEs[scNcgi][i+1:]...)
				break
			}
		}

		for index, ncgi := range m.UEToServingCells[ueIMSI] {
			if ncgi == scNcgi {
				m.UEToServingCells[ueIMSI] = append(m.UEToServingCells[ueIMSI][:index], m.UEToServingCells[ueIMSI][index+1:]...)
				break
			}
		}

	}

	// append ue to targetCells &  targetCell to ue
	for _, tcNcgi := range targetCellINcgis {
		m.CellToUEs[tcNcgi] = append(m.CellToUEs[tcNcgi], ueIMSI)
		m.UEToServingCells[ueIMSI] = append(m.UEToServingCells[ueIMSI], tcNcgi)
	}

	// LogUECells(ue)
	m.UpdateUECells(sourceCellNcgis, targetCellINcgis, ue)
	// LogUECells(ue)

}

func LogUECells(ue *UE) {
	sCellNCGIs := []types.NCGI{}
	nCellNCGIs := []types.NCGI{}
	for _, ueServCell := range ue.ServingCells {
		if ueServCell == nil {
			logrus.Errorf("ue: %v | nil serving cell found!", ue.IMSI)
		}
		sCellNCGIs = append(sCellNCGIs, ueServCell.NCGI)
	}
	for _, ueNeighCell := range ue.NeighborCells {
		nCellNCGIs = append(nCellNCGIs, ueNeighCell.NCGI)
	}
	logrus.Infof(
		`ue: %v | ue.ServingCells:%+v ue.NeighborCells:%+v `,
		ue.IMSI, sCellNCGIs, nCellNCGIs,
	)
}

// UpdateUECells updates the serving and neighbor cells pointed by the ue.
func (m *Model) UpdateUECells(sourceCellNcgis, targetCellINcgis []types.NCGI, ue *UE) {

	deletedUECells := []*UECell{}
	// for each serving cell remove from serving and add to neighbors.
SERVING_CELL_DELETION:
	for _, servCellNCGI := range sourceCellNcgis {
		for _, targetUECellINcgi := range targetCellINcgis {
			if servCellNCGI == targetUECellINcgi {
				continue SERVING_CELL_DELETION
			}
		}
		deletedUECell := ue.DeleteServingCell(servCellNCGI)
		if deletedUECell == nil {
			continue
		}
		logrus.Infof("ue: %v | deleting serving cell: %v", ue.IMSI, deletedUECell.NCGI)
		deletedUECells = append(deletedUECells, deletedUECell)
	}

	// for each target serving cell
	// remove from neighbor and add to serving cells
	for _, tCellNCGI := range targetCellINcgis {
		neighUECellIndex, neighTargetUECell := ue.GetNeighborCell(tCellNCGI)
		if neighTargetUECell == nil {
			continue
		}
		ue.NeighborCells = append(ue.NeighborCells[neighUECellIndex:], ue.NeighborCells[neighUECellIndex+1:]...)
		ue.ServingCells = append(ue.ServingCells, neighTargetUECell)
	}
	ue.NeighborCells = append(ue.NeighborCells, deletedUECells...)

}

func (m *Model) InitServiceMappings(ues map[string]*UE) {
	m.CellToUEs = make(map[types.NCGI][]types.IMSI)
	m.UEToServingCells = make(map[types.IMSI][]types.NCGI)

	for _, ue := range ues {
		for _, ueSCell := range ue.ServingCells {
			ueIMSI := ue.IMSI
			if _, exists := m.CellToUEs[ueSCell.NCGI]; !exists {
				m.CellToUEs[ueSCell.NCGI] = []types.IMSI{}
			}
			m.CellToUEs[ueSCell.NCGI] = append(m.CellToUEs[ueSCell.NCGI], ueIMSI)

			if _, exists := m.UEToServingCells[ueIMSI]; !exists {
				m.UEToServingCells[ueIMSI] = []types.NCGI{}
			}
			m.UEToServingCells[ueIMSI] = append(m.UEToServingCells[ueIMSI], ueSCell.NCGI)
		}

	}
}

func (m *Model) GetServedUEs(ncgi types.NCGI) []*UE {
	servedUEs := []*UE{}
	for _, imsi := range m.CellToUEs[ncgi] {
		imsiStr := strconv.Itoa(int(imsi))
		ue := m.UEs[imsiStr]
		servedUEs = append(servedUEs, ue)
	}
	return servedUEs
}

func (m *Model) GetServingCells(imsi types.IMSI) []*Cell {
	servingCells := []*Cell{}
	for _, ncgi := range m.UEToServingCells[imsi] {
		ncgiStr := strconv.Itoa(int(ncgi))
		cell := m.Cells[ncgiStr]
		servingCells = append(servingCells, cell)
	}
	return servingCells
}

func (m *Model) GetCarrier(beamID BeamID) *Carrier {
	sCell := m.Cells[strconv.FormatUint(uint64(beamID.NCGI), 10)]
	return sCell.Carriers[beamID.CarrierIndex-1]
}

func (m *Model) GetBeam(beamID BeamID) *Beam {
	sCell := m.Cells[strconv.FormatUint(uint64(beamID.NCGI), 10)]
	return sCell.Carriers[beamID.CarrierIndex-1].Beams[beamID.BeamIndex-1]
}

type ServiceMappings struct {
	sync.RWMutex
	CellToUEs        map[types.NCGI][]types.IMSI `mapstructure:"cellToUEs" yaml:"cellToUEs"`
	UEToServingCells map[types.IMSI][]types.NCGI `mapstructure:"ueToServingCells" yaml:"ueToServingCells"`
}

// Coordinate represents a geographical location
type Coordinate struct {
	Lat float64 `mapstructure:"lat" yaml:"lat"`
	Lng float64 `mapstructure:"lng" yaml:"lng"`
}

// RouteEndPoint ...
type RouteEndPoint struct {
	Start Coordinate `mapstructure:"start" yaml:"start"`
	End   Coordinate `mapstructure:"end" yaml:"end"`
}

// Route represents a series of points for tracking movement of user-equipment
type Route struct {
	IMSI        types.IMSI    `mapstructure:"imsi" yaml:"imsi"`
	Points      []*Coordinate `mapstructure:"points" yaml:"points"`
	Color       string        `mapstructure:"color" yaml:"color"`
	SpeedAvg    uint32        `mapstructure:"speedAvg" yaml:"speedAvg"`
	SpeedStdDev uint32        `mapstructure:"speedStdDev" yaml:"speedStdDev"`
	Reverse     bool          `mapstructure:"reverse" yaml:"reverse"`
	NextPoint   uint32        `mapstructure:"nextPoint" yaml:"nextPoint"`
}

// Node e2 node
type Node struct {
	GnbID         types.GnbID  `mapstructure:"gnbid" yaml:"gnbid"`
	Controllers   []string     `mapstructure:"controllers" yaml:"controllers"`
	ServiceModels []string     `mapstructure:"servicemodels" yaml:"servicemodels"`
	Cells         []types.NCGI `mapstructure:"cells" yaml:"cells"`
	Status        string       `mapstructure:"status" yaml:"status"`
}

// Controller E2T endpoint information
type Controller struct {
	ID      string `mapstructure:"id" yaml:"id"`
	Address string `mapstructure:"address" yaml:"address"`
	Port    int    `mapstructure:"port" yaml:"port"`
}

// MeasurementParams has measurement parameters
type MeasurementParams struct {
	TimeToTrigger          int32                `mapstructure:"timeToTrigger" yaml:"timeToTrigger"`
	FrequencyOffset        int32                `mapstructure:"frequencyOffset" yaml:"frequencyOffset"`
	PCellIndividualOffset  int32                `mapstructure:"pcellIndividualOffset" yaml:"pcellIndividualOffset"`
	NCellIndividualOffsets map[types.NCGI]int32 `mapstructure:"ncellIndividualOffsets" yaml:"ncellIndividualOffsets"`
	Hysteresis             int32                `mapstructure:"hysteresis" yaml:"hysteresis"`
	EventA3Params          EventA3Params        `mapstructure:"eventA3Params" yaml:"eventA3Params"`
}

// EventA3Params has event a3 parameters
type EventA3Params struct {
	A3Offset      int32 `mapstructure:"a3Offset" yaml:"a3Offset"`
	ReportOnLeave bool  `mapstructure:"reportOnLeave" yaml:"reportOnLeave"`
}

// Guami is AMF ID
type Guami struct {
	AmfRegionID uint32 `mapstructure:"amfregionid" yaml:"amfregionid"`
	AmfSetID    uint32 `mapstructure:"amfsetid" yaml:"amfsetid"`
	AmfPointer  uint32 `mapstructure:"amfpointer" yaml:"amfpointer"`
}

type CellConfig struct {
	Carriers []*Carrier `mapstructure:"carriers" yaml:"carriers"`
	SchedulingCellInfo
	RATType
}

type CellCoverageInfo struct {
	RPCoverageBoundaries map[BeamID][]CoverageBoundary `mapstructure:"rpCoverageBoundaries" yaml:"rpCoverageBoundaries"`
	CoverageBoundaries   map[BeamID][]CoverageBoundary `mapstructure:"coverageBoundaries" yaml:"coverageBoundaries"`
}

type SchedulingCellInfo string

const (
	SCHEDULING_CELL_INFO_OWN   SchedulingCellInfo = "own"
	SCHEDULING_CELL_INFO_OTHER SchedulingCellInfo = "other"
)

type RATType string

const (
	RAT_NR    RATType = "NR"
	RAT_EUTRA RATType = "EUTRA"
)

// Cell represents a section of coverage
type Cell struct {
	sync.RWMutex
	CellConfig
	NCGI                types.NCGI                   `mapstructure:"ncgi" yaml:"ncgi"`
	Color               string                       `mapstructure:"color" yaml:"color"`
	MaxUEs              uint32                       `mapstructure:"maxUEs" yaml:"maxUEs"`
	Neighbors           []types.NCGI                 `mapstructure:"neighbors" yaml:"neighbors"`
	MeasurementParams   MeasurementParams            `mapstructure:"measurementParams" yaml:"measurementParams"`
	PCI                 uint32                       `mapstructure:"pci" yaml:"pci"`
	Earfcn              uint32                       `mapstructure:"earfcn" yaml:"earfcn"`
	CellType            types.CellType               `mapstructure:"cellType" yaml:"cellType"`
	ArfcnDL             uint32                       `mapstructure:"arfcndl"`
	ArfcnUL             uint32                       `mapstructure:"arfcnul"`
	EarfcnDL            uint32                       `mapstructure:"earfcndl"`
	EarfcnUL            uint32                       `mapstructure:"earfcnul"`
	BsChannelBwDL       uint32                       `mapstructure:"bSChannelBwDL" yaml:"bSChannelBwDL"`
	BsChannelBwUL       uint32                       `mapstructure:"bSChannelBwUL" yaml:"bSChannelBwUL"`
	Bwps                map[uint64]*Bwp              `mapstructure:"bwps" yaml:"bwps"`
	RrcIdleCount        uint32                       `mapstructure:"rrcIdleCount" yaml:"rrcIdleCount"`
	RrcConnectedCount   uint32                       `mapstructure:"rrcConnectedCount" yaml:"rrcConnectedCount"`
	Cached              bool                         `mapstructure:"cached" yaml:"cached"`
	CachedStates        map[string]*CellCoverageInfo `mapstructure:"cachedStates" yaml:"cachedStates"`
	CurrentStateHash    string                       `mapstructure:"currentStateHash" yaml:"currentStateHash"`
	ResourceAllocScheme string                       `mapstructure:"resourceAllocScheme" yaml:"resourceAllocScheme"`
	InterferingBeams    map[BeamID][]BeamID          `mapstructure:"interfearingBeamsrefs" yaml:"interfearingBeamsrefs"`
	Grid
}

func (cell *Cell) GetCellConfig() CellConfig {
	return CellConfig{Carriers: cell.Carriers}
}

func (cell *Cell) GetHashedConfig() string {
	cellConfig, _ := json.Marshal(cell.GetCellConfig())
	hash := sha256.New()
	hash.Write(cellConfig)

	return hex.EncodeToString(hash.Sum(nil))
}

func (cell *Cell) GetBeam(beamID BeamID) *Beam {
	return cell.Carriers[beamID.CarrierIndex-1].Beams[beamID.BeamIndex-1]
}

func (cell *Cell) GetCarrier(beamID BeamID) *Carrier {
	return cell.Carriers[beamID.CarrierIndex-1]
}

type Carrier struct {
	Beams                  []*Beam    `mapstructure:"beams" yaml:"beams" json:"beams"`
	Center                 Coordinate `mapstructure:"center" yaml:"center" json:"center"`
	Height                 int32      `mapstructure:"height" yaml:"height" json:"height"`
	ArfcnDL                uint32     `mapstructure:"arfcndl" yaml:"arfcndl" json:"arfcndl"`
	ArfcnUL                uint32     `mapstructure:"arfcnul" yaml:"arfcnul" json:"arfcnul"`
	BsChannelBwDL          uint32     `mapstructure:"bSChannelBwDL" yaml:"bSChannelBwDL" json:"bSChannelBwDL"`
	BsChannelBwUL          uint32     `mapstructure:"bSChannelBwUL" yaml:"bSChannelBwUL" json:"bSChannelBwUL"`
	TxPowerDB              float64    `mapstructure:"txpowerdb" yaml:"txpowerdb" json:"txpowerdb"`
	VSideLobeAttenuationDB float64    `mapstructure:"vSideLobeAttenuationDB" yaml:"vSideLobeAttenuationDB" json:"vSideLobeAttenuationDB"`
	Environment            string     `mapstructure:"environment" yaml:"environment" json:"environment" validate:"oneof=urban rural"`
	LOS                    bool       `mapstructure:"LOS" yaml:"LOS" json:"LOS"`
}

type BeamID struct {
	NCGI         types.NCGI `mapstructure:"ncgi" yaml:"ncgi" json:"ncgi"`
	CarrierIndex int        `mapstructure:"carrierIndex" yaml:"carrierIndex" json:"carrierIndex"`
	BeamIndex    int        `mapstructure:"beamIndex" yaml:"beamIndex" json:"beamIndex"`
}

func (beamID BeamID) ToString() string {
	return fmt.Sprintf("%d_%d_%d", beamID.NCGI, beamID.CarrierIndex, beamID.BeamIndex)
}

func ParseBeamID(key string) (BeamID, error) {
	parts := strings.Split(key, "_")
	if len(parts) != 3 {
		return BeamID{}, fmt.Errorf("invalid BeamID key: %s", key)
	}
	ncgi, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return BeamID{}, err
	}
	carrierIndex, err := strconv.Atoi(parts[1])
	if err != nil {
		return BeamID{}, err
	}
	beamIndex, err := strconv.Atoi(parts[2])
	if err != nil {
		return BeamID{}, err
	}
	return BeamID{NCGI: types.NCGI(ncgi), CarrierIndex: carrierIndex, BeamIndex: beamIndex}, nil
}

type BeamQS struct {
	BeamID BeamID `mapstructure:"beamID" yaml:"beamID" json:"beamID"`
	CQI    int    `mapstructure:"cqi" yaml:"cqi" json:"cqi"`
}

type Beam struct {
	BeamIndex int     `mapstructure:"beamIndex" yaml:"beamIndex"`
	Azimuth   float64 `mapstructure:"azimuth" yaml:"azimuth"`
	Tilt      float64 `mapstructure:"tilt" yaml:"tilt"`
	H3dBAngle float64 `mapstructure:"h3dBAngle" yaml:"h3dBAngle"` // BeamHorizWidth
	V3dBAngle float64 `mapstructure:"v3dBAngle" yaml:"v3dBAngle"` // BeamVertWidth
	MaxGain   float64 `mapstructure:"maxGain" yaml:"maxGain"`
}

type Grid struct {
	ShadowingMaps map[BeamID][]float64    `mapstructure:"shadowingMaps" yaml:"shadowingMaps" json:"shadowingMaps"`
	GridPoints    map[BeamID][]Coordinate `mapstructure:"gridPoints" yaml:"gridPoints" json:"gridPoints"`
	BoundingBoxes map[BeamID]*BoundingBox `mapstructure:"boundingBoxes" yaml:"boundingBoxes" json:"boundingBoxes"`
}

type BoundingBox struct {
	MinLat float64 `mapstructure:"minLat" yaml:"minLat" json:"minLat"`
	MinLng float64 `mapstructure:"minLng" yaml:"minLng" json:"minLng"`
	MaxLat float64 `mapstructure:"maxLat" yaml:"maxLat" json:"maxLat"`
	MaxLng float64 `mapstructure:"maxLng" yaml:"maxLng" json:"maxLng"`
}

func (bb *BoundingBox) GreaterThan(bb2 *BoundingBox) bool {
	bbArea := (bb.MaxLat - bb.MinLat) * (bb.MaxLng - bb.MinLng)
	bb2Area := (bb2.MaxLat - bb2.MinLat) * (bb2.MaxLng - bb2.MinLng)
	return bbArea > bb2Area
}

type CoverageBoundary struct {
	RefSignalStrength float64      `mapstructure:"refSignalStrength" yaml:"refSignalStrength" json:"refSignalStrength"`
	BoundaryPoints    []Coordinate `mapstructure:"boundaryPoints" yaml:"boundaryPoints" json:"boundaryPoints"`
}

// UEType represents type of user-equipment
type UEType string

// UECell represents UE-cell relationship
type UECell struct {
	ID          types.GnbID `mapstructure:"id" yaml:"id" json:"id"`
	NCGI        types.NCGI  `mapstructure:"ncgi" yaml:"ncgi" json:"ncgi"` // Auxiliary form of association
	BeamID      BeamID      `mapstructure:"beamID" yaml:"beamID" json:"beamID"`
	Rsrp        float64     `mapstructure:"rsrp" yaml:"rsrp" json:"rsrp"`
	Rsrq        float64     `mapstructure:"rsrq" yaml:"rsrq" json:"rsrq"`
	Sinr        float64     `mapstructure:"sinr" yaml:"sinr" json:"sinr"`
	BwpRefs     []*Bwp      `mapstructure:"bwpRefs" yaml:"bwpRefs" json:"bwpRefs"`
	AvailPrbsDl int         `mapstructure:"availPrbsDl" yaml:"availPrbsDl" json:"availPrbsDl"`
}

type Bwp struct {
	ID          uint64 `mapstructure:"id" yaml:"id" json:"id"`
	Scs         int    `mapstructure:"scs" yaml:"scs" json:"scs"`
	NumberOfRBs int    `mapstructure:"numberOfRBs" yaml:"numberOfRBs" json:"numberOfRBs"`
	Downlink    bool   `mapstructure:"downlink" yaml:"downlink" json:"downlink"`
}

// UE represents user-equipment, i.e. phone, IoT device, etc.
type UE struct {
	IMSI                      types.IMSI                                `mapstructure:"imsi" yaml:"imsi" json:"imsi"`
	AmfUeNgapID               types.AmfUENgapID                         `mapstructure:"amfUeNgapID" yaml:"amfUeNgapID" json:"amfUeNgapID"`
	Type                      UEType                                    `mapstructure:"type" yaml:"type" json:"type"`
	RrcState                  e2sm_mho.Rrcstatus                        `mapstructure:"rrcState" yaml:"rrcState" json:"rrcState"`
	Location                  Coordinate                                `mapstructure:"location" yaml:"location" json:"location"`
	Heading                   uint32                                    `mapstructure:"heading" yaml:"heading" json:"heading"`
	FiveQi                    int                                       `mapstructure:"fiveQi" yaml:"fiveQi" json:"fiveQi"`
	ServingCells              []*UECell                                 `mapstructure:"servingCells" yaml:"servingCells" json:"servingCells"`
	NeighborCells             []*UECell                                 `mapstructure:"neighborCells" yaml:"neighborCells" json:"neighborCells"`
	InterferingBeams          []*UECell                                 `mapstructure:"interferingBeams" yaml:"interferingBeams" json:"interferingBeams"`
	CRNTI                     types.CRNTI                               `mapstructure:"CRNTI" yaml:"CRNTI" json:"CRNTI"`
	Height                    float64                                   `mapstructure:"height" yaml:"height" json:"height"`
	IsAdmitted                bool                                      `mapstructure:"isAdmitted" yaml:"isAdmitted" json:"isAdmitted"`
	SupportedBandCombinations map[ConnectivityType]*ConnTypeSupportInfo `mapstructure:"supportedBandCombinations"`
	SupportedBandsNR          []string                                  `mapstructure:"supportedBandsNR"`
	SupportedBandsEutra       []string                                  `mapstructure:"supportedBandsEutra"`
}

func (ue *UE) GetServingCell(ncgi types.NCGI) (*UECell, bool) {
	for servCellIndex := range ue.ServingCells {
		if ncgi == ue.ServingCells[servCellIndex].NCGI {
			return ue.ServingCells[servCellIndex], true
		}
	}
	return nil, false
}

func (ue *UE) DeleteServingCell(ncgi types.NCGI) *UECell {
	var deletedCell *UECell
	deletedCellIndex := -1
	for servCellIndex := range ue.ServingCells {
		if ncgi == ue.ServingCells[servCellIndex].NCGI {
			deletedCellIndex = servCellIndex
			deletedCell = ue.ServingCells[servCellIndex]
			break
		}
	}
	if deletedCellIndex == -1 {
		return nil
	}
	ue.ServingCells = append(ue.ServingCells[:deletedCellIndex], ue.ServingCells[deletedCellIndex+1:]...)
	return deletedCell
}

func (ue *UE) GetNeighborCell(ncgi types.NCGI) (int, *UECell) {
	for neighCellIndex := range ue.NeighborCells {
		if ncgi == ue.NeighborCells[neighCellIndex].NCGI {
			return neighCellIndex, ue.NeighborCells[neighCellIndex]
		}
	}
	return -1, nil
}

type ConnectivityType string

const (
	NR    ConnectivityType = "NR"
	EUTRA ConnectivityType = "EUTRA"
	EN_DC ConnectivityType = "EN-DC"
	NR_DC ConnectivityType = "NR-DC"
	NE_DC ConnectivityType = "NE-DC"
)

type ConnTypeSupportInfo struct {
	SupportedBandCombinations []*BandCombination
}

type BandSupportInfo struct {
	Band           string
	BandwidthClass string
	MIMOLayers     string
}

type BandCombination struct {
	Direction         string // UL/DL
	CombinedBandsInfo []*BandSupportInfo
}

// ServiceModel service model information
type ServiceModel struct {
	ID          int    `mapstructure:"id" yaml:"id" json:"id"`
	Description string `mapstructure:"description" yaml:"description" json:"description"`
	Version     string `mapstructure:"version" yaml:"version" json:"version"`
}

// GetServiceModel gets a service model based on a given name.
func (m *Model) GetServiceModel(name string) (ServiceModel, error) {
	if sm, ok := m.ServiceModels[name]; ok {
		return sm, nil
	}
	return ServiceModel{}, errors.New(errors.NotFound, "the service model not found")
}

// GetController gets a controller by a given name
func (m *Model) GetController(name string) (Controller, error) {
	if controller, ok := m.Controllers[name]; ok {
		return controller, nil
	}
	return Controller{}, errors.New(errors.NotFound, "controller not found")
}
