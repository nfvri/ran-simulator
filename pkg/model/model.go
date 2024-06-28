// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"
	"github.com/onosproject/onos-lib-go/pkg/errors"
)

// Model simulation model
type Model struct {
	MapLayout               MapLayout               `mapstructure:"layout" yaml:"layout"`
	RouteEndPoints          []RouteEndPoint         `mapstructure:"routeEndPoints" yaml:"routeEndPoints"`
	WayPointRoute           bool                    `mapstructure:"wayPointRoute" yaml:"wayPointRoute"`
	DirectRoute             bool                    `mapstructure:"directRoute" yaml:"directRoute"`
	Nodes                   map[string]Node         `mapstructure:"nodes" yaml:"nodes"`
	Cells                   map[string]Cell         `mapstructure:"cells" yaml:"cells"`
	Controllers             map[string]Controller   `mapstructure:"controllers" yaml:"controllers"`
	ServiceModels           map[string]ServiceModel `mapstructure:"servicemodels" yaml:"servicemodels"`
	RrcStateChangesDisabled bool                    `mapstructure:"RrcStateChangesDisabled" yaml:"RrcStateChangesDisabled"`
	InitialRrcState         string                  `mapstructure:"initialRrcState" yaml:"initialRrcState"`
	UECount                 uint                    `mapstructure:"ueCount" yaml:"ueCount"`
	UECountPerCell          uint                    `mapstructure:"ueCountPerCell" yaml:"ueCountPerCell"`
	UEList                  map[string]UE           `mapstructure:"ues" yaml:"ues"`
	Plmn                    string                  `mapstructure:"plmnID" yaml:"plmnID"`
	PlmnID                  types.PlmnID            `mapstructure:"plmnNumber" yaml:"plmnNumber"` // overridden and derived post-load from "Plmn" field
	APIKey                  string                  `mapstructure:"apiKey" yaml:"apiKey"`         // Google Maps API key (optional)
	Guami                   Guami                   `mapstructure:"guami" yaml:"guami"`
	DecorrelationDistance   float64                 `mapstructure:"decorrelationdistance"`
}

// Coordinate represents a geographical location
type Coordinate struct {
	Lat float64 `mapstructure:"lat"`
	Lng float64 `mapstructure:"lng"`
}

// Sector represents a 2D arc emanating from a location
type Sector struct {
	Center  Coordinate `mapstructure:"center"`
	Azimuth float64    `mapstructure:"azimuth"`
	Arc     int32      `mapstructure:"arc"`
	Tilt    float64    `mapstructure:"tilt"`
	Height  int32      `mapstructure:"height"`
}

// RouteEndPoint ...
type RouteEndPoint struct {
	Start Coordinate `mapstructure:"start"`
	End   Coordinate `mapstructure:"end"`
}

// Route represents a series of points for tracking movement of user-equipment
type Route struct {
	IMSI        types.IMSI
	Points      []*Coordinate
	Color       string
	SpeedAvg    uint32
	SpeedStdDev uint32
	Reverse     bool
	NextPoint   uint32
}

// Node e2 node
type Node struct {
	GnbID         types.GnbID  `mapstructure:"gnbid"`
	Controllers   []string     `mapstructure:"controllers"`
	ServiceModels []string     `mapstructure:"servicemodels"`
	Cells         []types.NCGI `mapstructure:"cells"`
	Status        string       `mapstructure:"status"`
}

// Controller E2T endpoint information
type Controller struct {
	ID      string `mapstructure:"id"`
	Address string `mapstructure:"address"`
	Port    int    `mapstructure:"port"`
}

// MeasurementParams has measurement parameters
type MeasurementParams struct {
	TimeToTrigger          int32                `mapstructure:"timeToTrigger"`
	FrequencyOffset        int32                `mapstructure:"frequencyOffset"`
	PCellIndividualOffset  int32                `mapstructure:"pcellIndividualOffset"`
	NCellIndividualOffsets map[types.NCGI]int32 `mapstructure:"ncellIndividualOffsets"`
	Hysteresis             int32                `mapstructure:"hysteresis"`
	EventA3Params          EventA3Params        `mapstructure:"eventA3Params"`
}

// EventA3Params has event a3 parameters
type EventA3Params struct {
	A3Offset      int32 `mapstructure:"a3Offset"`
	ReportOnLeave bool  `mapstructure:"reportOnLeave"`
}

// Guami is AMF ID
type Guami struct {
	AmfRegionID uint32 `mapstructure:"amfregionid"`
	AmfSetID    uint32 `mapstructure:"amfsetid"`
	AmfPointer  uint32 `mapstructure:"amfpointer"`
}

// Channel represents a sl sector operational frequency range
type Channel struct {
	SSBFrequency uint32 `mapstructure:"ssbfrequency"`
	ArfcnDL      uint32 `mapstructure:"arfcndl"`
	ArfcnUL      uint32 `mapstructure:"arfcnul"`
	Environment  string `mapstructure:"environment" validate:"oneof=urban rural"`
	LOS          bool   `mapstructure:"LOS"`
}

// Cell represents a section of coverage
type Cell struct {
	NCGI               types.NCGI         `mapstructure:"ncgi"`
	Sector             Sector             `mapstructure:"sector"`
	Color              string             `mapstructure:"color"`
	MaxUEs             uint32             `mapstructure:"maxUEs"`
	Neighbors          []types.NCGI       `mapstructure:"neighbors"`
	TxPowerDB          float64            `mapstructure:"txpowerdb"`
	MeasurementParams  MeasurementParams  `mapstructure:"measurementParams"`
	PCI                uint32             `mapstructure:"pci"`
	Earfcn             uint32             `mapstructure:"earfcn"`
	CellType           types.CellType     `mapstructure:"cellType"`
	Channel            Channel            `mapstructure:"channel"`
	Beam               Beam               `mapstructure:"beam"`
	CoverageBoundaries []CoverageBoundary `mapstructure:"coverageBoundaries"`
	RrcIdleCount       uint32
	RrcConnectedCount  uint32
	ShadowMap
}

type Beam struct {
	H3dBAngle              float64 `mapstructure:"h3dBAngle"`
	V3dBAngle              float64 `mapstructure:"v3dBAngle"`
	MaxGain                float64 `mapstructure:"maxGain"`
	MaxAttenuationDB       float64 `mapstructure:"maxAttenuationDB"`
	VSideLobeAttenuationDB float64 `mapstructure:"vSideLobeAttenuationDB"`
}

type ShadowMap struct {
	ShadowingMap [][]float64  `json:"shadowingMap"`
	GridPoints   []Coordinate `json:"gridPoints"`
}

type CoverageBoundary struct {
	RefSignalStrength int          `json:"refSignalStrength"`
	BoundaryPoints    []Coordinate `json:"boundaryPoints"`
}

// UEType represents type of user-equipment
type UEType string

// UECell represents UE-cell relationship
type UECell struct {
	ID   types.GnbID `mapstructure:"id"`
	NCGI types.NCGI  `mapstructure:"ncgi"` // Auxiliary form of association
	Rsrp float64     `mapstructure:"rsrp"`
	Rsrq float64     `mapstructure:"rsrq"`
	Sinr float64     `mapstructure:"sinr"`
}

// UE represents user-equipment, i.e. phone, IoT device, etc.
type UE struct {
	IMSI        types.IMSI         `mapstructure:"imsi"`
	AmfUeNgapID types.AmfUENgapID  `mapstructure:"amfUeNgapID"`
	Type        UEType             `mapstructure:"type"`
	RrcState    e2sm_mho.Rrcstatus `mapstructure:"rrcState"`
	Location    Coordinate         `mapstructure:"location"`
	Heading     uint32             `mapstructure:"heading"`
	FiveQi      int                `mapstructure:"fiveQi"`
	Cell        *UECell            `mapstructure:"cell"`
	CRNTI       types.CRNTI        `mapstructure:"CRNTI"`
	Cells       []*UECell          `mapstructure:"cells"`
	Height      float64            `mapstructure:"height"`
	IsAdmitted  bool               `mapstructure:"isAdmitted"`
}

// ServiceModel service model information
type ServiceModel struct {
	ID          int    `mapstructure:"id"`
	Description string `mapstructure:"description"`
	Version     string `mapstructure:"version"`
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
