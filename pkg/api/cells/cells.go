// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package cells

import (
	"context"

	modelapi "github.com/nfvri/onos-api/go/onos/ransim/model"
	"github.com/nfvri/onos-api/go/onos/ransim/types"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/store/cells"
	"github.com/nfvri/ran-simulator/pkg/store/event"
	liblog "github.com/onosproject/onos-lib-go/pkg/logging"
	service "github.com/onosproject/onos-lib-go/pkg/northbound"
	"google.golang.org/grpc"
)

var log = liblog.GetLogger()

// NewService returns a new model Service
func NewService(cellStore cells.Store) service.Service {
	return &Service{
		cellStore: cellStore,
	}
}

// Service is a Service implementation for administration.
type Service struct {
	service.Service
	cellStore cells.Store
}

// Register registers the TrafficSim Service with the gRPC server.
func (s *Service) Register(r *grpc.Server) {
	server := &Server{
		cellStore: s.cellStore,
	}
	modelapi.RegisterCellModelServer(r, server)
}

// Server implements the TrafficSim gRPC service for administrative facilities.
type Server struct {
	cellStore cells.Store
}

func cellToAPI(cell *model.Cell) *types.Cell {

	return &types.Cell{
		CellConfig:          cellConfigToAPI(cell.GetCellConfig()),
		NCGI:                cell.NCGI,
		CellType:            cell.CellType,
		Color:               cell.Color,
		MaxUEs:              cell.MaxUEs,
		Neighbors:           cell.Neighbors,
		Earfcn:              cell.Earfcn,
		MeasurementParams:   measurementParamsToAPI(cell.MeasurementParams),
		RrcIdleCount:        cell.RrcIdleCount,
		RrcConnectedCount:   cell.RrcConnectedCount,
		Pci:                 cell.PCI,
		Cached:              cell.Cached,
		ResourceAllocScheme: cell.ResourceAllocScheme,
		CurrentStateHash:    cell.CurrentStateHash,
		Grid:                gridToAPI(cell.Grid),
		Bwps:                bwpsToAPI(cell.Bwps),
		CachedStates:        cachedStatesToAPI(cell.CachedStates),
	}
}

func cellConfigToAPI(cellConfig model.CellConfig) *types.CellConfig {
	return &types.CellConfig{
		TxPowerdB: cellConfig.TxPowerDB,
		Sector:    sectorToAPI(cellConfig.Sector),
		Channel:   channelToAPI(cellConfig.Channel),
		Beam:      beamToAPI(cellConfig.Beam),
	}
}

func sectorToAPI(sector model.Sector) *types.Sector {
	return &types.Sector{
		Azimuth: float64(sector.Azimuth),
		Arc:     sector.Arc,
		Center:  (*types.Coordinate)(&sector.Center),
		Tilt:    float64(sector.Tilt),
		Height:  sector.Height,
	}
}

func channelToAPI(channel model.Channel) *types.Channel {
	return &types.Channel{
		SsbFrequency:   channel.SSBFrequency,
		ArfcnDl:        channel.ArfcnDL,
		ArfcnUl:        channel.ArfcnUL,
		Environment:    channel.Environment,
		BsChannelBwDl:  channel.BsChannelBwDL,
		BsChannelBwUl:  channel.BsChannelBwUL,
		BsChannelBwSul: channel.BsChannelBwSUL,
		Los:            channel.LOS,
	}
}

func beamToAPI(beam model.Beam) *types.Beam {
	return &types.Beam{
		H3DbAngle:              beam.H3dBAngle,
		V3DbAngle:              beam.V3dBAngle,
		MaxGain:                beam.MaxGain,
		MaxAttenuationDb:       beam.MaxAttenuationDB,
		VSideLobeAttenuationDb: beam.VSideLobeAttenuationDB,
	}
}

func gridToAPI(grid model.Grid) *types.Grid {
	return &types.Grid{
		ShadowingMap: grid.ShadowingMap,
		GridPoints:   sliceCoordToAPI(grid.GridPoints),
		BoundingBox:  (*types.BoundingBox)(grid.BoundingBox),
	}
}

func sliceCoordToAPI(modelGridPoints []model.Coordinate) []*types.Coordinate {
	gridPoints := make([]*types.Coordinate, len(modelGridPoints))
	for i, modelGridPoint := range modelGridPoints {
		gridPoints[i] = (*types.Coordinate)(&modelGridPoint)
	}
	return gridPoints
}

func bwpsToAPI(modelBWPs map[uint64]*model.Bwp) map[uint64]*types.Bwp {
	bwps := make(map[uint64]*types.Bwp, len(modelBWPs))
	for key, bwp := range modelBWPs {
		bwps[key] = &types.Bwp{
			Id:          bwp.ID,
			Scs:         int32(bwp.Scs),
			NumberOfRbs: int32(bwp.NumberOfRBs),
			Downlink:    bwp.Downlink,
		}
	}
	return bwps
}

func cachedStatesToAPI(modelCachedStates map[string]*model.CellSignalInfo) map[string]*types.CellSignalInfo {
	cachedStates := make(map[string]*types.CellSignalInfo, len(modelCachedStates))
	for key, modelCellSignalInfo := range modelCachedStates {
		cachedStates[key] = &types.CellSignalInfo{
			RpCoverageBoundaries: coverageBoundariesToAPI(modelCellSignalInfo.CoverageBoundaries),
			CoverageBoundaries:   coverageBoundariesToAPI(modelCellSignalInfo.CoverageBoundaries),
		}
	}
	return cachedStates
}

func coverageBoundariesToAPI(modelCoverageBoundaries []model.CoverageBoundary) []*types.CoverageBoundary {
	coverageBoundaries := make([]*types.CoverageBoundary, len(modelCoverageBoundaries))
	for i, covBoundary := range modelCoverageBoundaries {
		coverageBoundaries[i] = &types.CoverageBoundary{
			RefSignalStrength: covBoundary.RefSignalStrength,
			BoundaryPoints:    sliceCoordToAPI(covBoundary.BoundaryPoints),
		}
	}
	return coverageBoundaries
}

func measurementParamsToAPI(params model.MeasurementParams) *types.MeasurementParams {
	return &types.MeasurementParams{
		TimeToTrigger:          params.TimeToTrigger,
		FrequencyOffset:        params.FrequencyOffset,
		PcellIndividualOffset:  params.PCellIndividualOffset,
		NcellIndividualOffsets: params.NCellIndividualOffsets,
		Hysteresis:             params.Hysteresis,
		EventA3Params:          eventA3ParamsToAPI(params.EventA3Params),
	}
}

func eventA3ParamsToAPI(params model.EventA3Params) *types.EventA3Params {
	return &types.EventA3Params{
		A3Offset:      params.A3Offset,
		ReportOnLeave: params.ReportOnLeave,
	}
}

func cellToModel(cell *types.Cell) *model.Cell {
	cellConfig := cell.CellConfig
	cellSector := cellConfig.Sector
	cellBeam := cellConfig.Beam
	cellChannel := cellConfig.Channel
	return &model.Cell{
		CellConfig: model.CellConfig{
			TxPowerDB: cellConfig.TxPowerdB,
			Sector: model.Sector{
				Center:  model.Coordinate{Lat: cellSector.Center.Lat, Lng: cellSector.Center.Lng},
				Arc:     cellSector.Arc,
				Azimuth: float64(cellSector.Azimuth),
				Tilt:    float64(cellSector.Tilt),
				Height:  cellSector.Height,
			},
			Channel: model.Channel{
				SSBFrequency:   cellChannel.SsbFrequency,
				ArfcnDL:        cellChannel.ArfcnDl,
				ArfcnUL:        cellChannel.ArfcnUl,
				Environment:    cellChannel.Environment,
				BsChannelBwDL:  cellChannel.BsChannelBwDl,
				BsChannelBwUL:  cellChannel.BsChannelBwUl,
				BsChannelBwSUL: cellChannel.BsChannelBwSul,
				LOS:            cellChannel.Los,
			},
			Beam: model.Beam{
				H3dBAngle:              cellBeam.H3DbAngle,
				V3dBAngle:              cellBeam.V3DbAngle,
				MaxGain:                cellBeam.MaxGain,
				MaxAttenuationDB:       cellBeam.MaxAttenuationDb,
				VSideLobeAttenuationDB: cellBeam.VSideLobeAttenuationDb,
			},
		},
		NCGI:      cell.NCGI,
		CellType:  cell.CellType,
		Color:     cell.Color,
		MaxUEs:    cell.MaxUEs,
		Neighbors: cell.Neighbors,
		Earfcn:    cell.Earfcn,
		MeasurementParams: model.MeasurementParams{
			TimeToTrigger:          cell.MeasurementParams.TimeToTrigger,
			FrequencyOffset:        cell.MeasurementParams.FrequencyOffset,
			PCellIndividualOffset:  cell.MeasurementParams.PcellIndividualOffset,
			NCellIndividualOffsets: cell.MeasurementParams.NcellIndividualOffsets,
			Hysteresis:             cell.MeasurementParams.Hysteresis,
			EventA3Params: model.EventA3Params{
				A3Offset:      cell.MeasurementParams.EventA3Params.A3Offset,
				ReportOnLeave: cell.MeasurementParams.EventA3Params.ReportOnLeave,
			},
		},
		RrcIdleCount:        cell.RrcIdleCount,
		RrcConnectedCount:   cell.RrcConnectedCount,
		PCI:                 cell.Pci,
		Cached:              cell.Cached,
		ResourceAllocScheme: cell.ResourceAllocScheme,
		CurrentStateHash:    cell.CurrentStateHash,
		Grid: model.Grid{
			ShadowingMap: cell.Grid.ShadowingMap,
			GridPoints:   sliceCoordToModel(cell.Grid.GridPoints),
			BoundingBox:  (*model.BoundingBox)(cell.Grid.BoundingBox),
		},
		Bwps:         bwpsToModel(cell.Bwps),
		CachedStates: cachedStatesToModel(cell.CachedStates),
	}
}

func sliceCoordToModel(gridPoints []*types.Coordinate) []model.Coordinate {
	modelGridPoints := make([]model.Coordinate, len(gridPoints))
	for i, gridPoint := range gridPoints {
		modelGridPoints[i] = model.Coordinate(*gridPoint)
	}
	return modelGridPoints
}

func bwpsToModel(bwps map[uint64]*types.Bwp) map[uint64]*model.Bwp {
	modelBWPs := make(map[uint64]*model.Bwp, len(bwps))
	for key, bwp := range bwps {
		modelBWPs[key] = &model.Bwp{
			ID:          bwp.Id,
			Scs:         int(bwp.Scs),
			NumberOfRBs: int(bwp.NumberOfRbs),
			Downlink:    bwp.Downlink,
		}
	}
	return modelBWPs
}

func cachedStatesToModel(cachedStates map[string]*types.CellSignalInfo) map[string]*model.CellSignalInfo {
	modelCachedStates := make(map[string]*model.CellSignalInfo, len(cachedStates))
	for key, cellSignalInfo := range cachedStates {
		modelCachedStates[key] = &model.CellSignalInfo{
			RPCoverageBoundaries: coverageBoundariesToModel(cellSignalInfo.CoverageBoundaries),
			CoverageBoundaries:   coverageBoundariesToModel(cellSignalInfo.CoverageBoundaries),
		}
	}
	return modelCachedStates
}

func coverageBoundariesToModel(coverageBoundaries []*types.CoverageBoundary) []model.CoverageBoundary {
	modelCoverageBoundaries := make([]model.CoverageBoundary, len(coverageBoundaries))
	for i, covBoundary := range coverageBoundaries {
		modelCoverageBoundaries[i] = model.CoverageBoundary{
			RefSignalStrength: covBoundary.RefSignalStrength,
			BoundaryPoints:    sliceCoordToModel(covBoundary.BoundaryPoints),
		}
	}
	return modelCoverageBoundaries
}

// CreateCell creates a new simulated cell
func (s *Server) CreateCell(ctx context.Context, request *modelapi.CreateCellRequest) (*modelapi.CreateCellResponse, error) {
	log.Debugf("Received create cell request: %v", request)
	err := s.cellStore.Add(ctx, cellToModel(request.Cell))
	if err != nil {
		return nil, err
	}
	return &modelapi.CreateCellResponse{}, nil
}

// GetCell retrieves the specified simulated cell
func (s *Server) GetCell(ctx context.Context, request *modelapi.GetCellRequest) (*modelapi.GetCellResponse, error) {
	log.Debugf("Received get cell request: %v", request)
	node, err := s.cellStore.Get(ctx, request.NCGI)
	if err != nil {
		return nil, err
	}
	return &modelapi.GetCellResponse{Cell: cellToAPI(node)}, nil
}

// UpdateCell updates the specified simulated cell
func (s *Server) UpdateCell(ctx context.Context, request *modelapi.UpdateCellRequest) (*modelapi.UpdateCellResponse, error) {
	log.Debugf("Received update cell request: %v", request)
	err := s.cellStore.Update(ctx, cellToModel(request.Cell))
	if err != nil {
		return nil, err
	}
	return &modelapi.UpdateCellResponse{}, nil
}

// DeleteCell deletes the specified simulated cell
func (s *Server) DeleteCell(ctx context.Context, request *modelapi.DeleteCellRequest) (*modelapi.DeleteCellResponse, error) {
	log.Debugf("Received delete cell request: %v", request)
	_, err := s.cellStore.Delete(ctx, request.NCGI)
	if err != nil {
		return nil, err
	}
	return &modelapi.DeleteCellResponse{}, nil
}

func eventType(cellEvent cells.CellEvent) modelapi.EventType {
	if cellEvent == cells.Created {
		return modelapi.EventType_CREATED
	} else if cellEvent == cells.Updated {
		return modelapi.EventType_UPDATED
	} else if cellEvent == cells.Deleted {
		return modelapi.EventType_DELETED
	} else {
		return modelapi.EventType_NONE
	}
}

// ListCells list all of the cells
func (s *Server) ListCells(request *modelapi.ListCellsRequest, server modelapi.CellModel_ListCellsServer) error {
	log.Debugf("Received listing cells request: %v", request)
	cellList, err := s.cellStore.List(server.Context())
	if err != nil {
		return err
	}
	for _, cell := range cellList {
		resp := &modelapi.ListCellsResponse{
			Cell: cellToAPI(cell),
		}
		err = server.Send(resp)
		if err != nil {
			return err
		}
	}
	return nil
}

// WatchCells monitors changes to the inventory of cells
func (s *Server) WatchCells(request *modelapi.WatchCellsRequest, server modelapi.CellModel_WatchCellsServer) error {
	log.Debugf("Received watching cell changes request: %v", request)
	ch := make(chan event.Event)
	err := s.cellStore.Watch(server.Context(), ch, cells.WatchOptions{Replay: !request.NoReplay, Monitor: !request.NoSubscribe})
	if err != nil {
		return err
	}

	for cellEvent := range ch {
		response := &modelapi.WatchCellsResponse{
			Cell: cellToAPI(cellEvent.Value.(*model.Cell)),
			Type: eventType(cellEvent.Type.(cells.CellEvent)),
		}
		err := server.Send(response)
		if err != nil {
			return err
		}
	}
	return nil
}
