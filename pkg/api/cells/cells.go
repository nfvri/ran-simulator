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
func NewService(cellStore cells.Store, patchedCells []model.Cell) service.Service {
	return &Service{
		cellStore:    cellStore,
		patchedCells: patchedCells,
	}
}

// Service is a Service implementation for administration.
type Service struct {
	service.Service
	cellStore    cells.Store
	patchedCells []model.Cell
}

// Register registers the TrafficSim Service with the gRPC server.
func (s *Service) Register(r *grpc.Server) {
	server := &Server{
		cellStore:    s.cellStore,
		patchedCells: s.patchedCells,
	}
	modelapi.RegisterCellModelServer(r, server)
}

// Server implements the TrafficSim gRPC service for administrative facilities.
type Server struct {
	cellStore    cells.Store
	patchedCells []model.Cell
}

func cellToAPI(cell *model.Cell) *types.Cell {

	return &types.Cell{
		CellConfig:          cellConfigToAPI(cell.GetCellConfig()),
		NCGI:                cell.NCGI,
		CellType:            cell.CellType,
		Color:               cell.Color,
		MaxUEs:              cell.MaxUEs,
		Neighbors:           cell.Neighbors,
		Earfcn:              cell.EarfcnDL,
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
		Carriers: carriersToAPI(cellConfig.Carriers),
	}
}

func carriersToAPI(modelCarriers []*model.Carrier) []*types.Carrier {
	carriers := []*types.Carrier{}
	for _, carrier := range modelCarriers {
		carriers = append(carriers, &types.Carrier{
			Beams:                  beamsToAPI(carrier.Beams),
			Center:                 (*types.Coordinate)(&carrier.Center),
			Height:                 carrier.Height,
			ArfcnDl:                carrier.ArfcnDL,
			ArfcnUl:                carrier.ArfcnUL,
			BsChannelBwDl:          carrier.BsChannelBwDL,
			BsChannelBwUl:          carrier.BsChannelBwUL,
			TxPowerdB:              carrier.TxPowerDB,
			VSideLobeAttenuationDb: carrier.VSideLobeAttenuationDB,
			Environment:            carrier.Environment,
			Los:                    carrier.LOS,
		})
	}

	return carriers
}

func beamsToAPI(modelBeams []*model.Beam) []*types.Beam {
	beams := []*types.Beam{}
	for _, beam := range modelBeams {
		beams = append(beams, &types.Beam{
			BeamIndex: int32(beam.BeamIndex),
			Azimuth:   beam.Azimuth,
			Tilt:      beam.Tilt,
			H3DbAngle: beam.H3dBAngle,
			V3DbAngle: beam.V3dBAngle,
			MaxGain:   beam.MaxGain,
		})
	}

	return beams
}

func gridToAPI(grid model.Grid) *types.Grid {
	return &types.Grid{
		ShadowingMaps:  shadowingMapsToAPI(grid.ShadowingMaps),
		GridPointsMaps: gridPointsToAPI(grid.GridPoints),
		BoundingBoxes:  boundingBoxesToAPI(grid.BoundingBoxes),
	}
}

func shadowingMapsToAPI(modelShadowingMaps map[model.BeamID][]float64) []*types.ShadowingMapEntry {
	shadowingMapEntries := make([]*types.ShadowingMapEntry, 0, len(modelShadowingMaps))
	for modelBeamID, shadowingMap := range modelShadowingMaps {
		beamID := types.BeamID{Ncgi: modelBeamID.NCGI, CarrierIndex: int32(modelBeamID.CarrierIndex), BeamIndex: int32(modelBeamID.BeamIndex)}
		shadowingMapEntries = append(shadowingMapEntries, &types.ShadowingMapEntry{
			BeamId:       (*types.BeamID)(&beamID),
			ShadowingMap: shadowingMap,
		})
	}
	return shadowingMapEntries
}

func gridPointsToAPI(modelGridPoints map[model.BeamID][]model.Coordinate) []*types.GridPointsEntry {
	gridPointsEntries := make([]*types.GridPointsEntry, 0, len(modelGridPoints))
	for modelBeamID, gridPoints := range modelGridPoints {
		beamID := types.BeamID{Ncgi: modelBeamID.NCGI, CarrierIndex: int32(modelBeamID.CarrierIndex), BeamIndex: int32(modelBeamID.BeamIndex)}
		gridPointsEntries = append(gridPointsEntries, &types.GridPointsEntry{
			BeamId:     (*types.BeamID)(&beamID),
			GridPoints: sliceCoordToAPI(gridPoints),
		})
	}
	return gridPointsEntries
}

func boundingBoxesToAPI(modelBoundingBoxes map[model.BeamID]*model.BoundingBox) []*types.BoundingBoxEntry {
	boundingBoxEntries := make([]*types.BoundingBoxEntry, 0, len(modelBoundingBoxes))
	for modelBeamID, boundingBox := range modelBoundingBoxes {
		beamID := types.BeamID{Ncgi: modelBeamID.NCGI, CarrierIndex: int32(modelBeamID.CarrierIndex), BeamIndex: int32(modelBeamID.BeamIndex)}
		boundingBoxEntries = append(boundingBoxEntries, &types.BoundingBoxEntry{
			BeamId:      (*types.BeamID)(&beamID),
			BoundingBox: (*types.BoundingBox)(boundingBox),
		})
	}
	return boundingBoxEntries
}

func sliceCoordToAPI(modelGridPoints []model.Coordinate) []*types.Coordinate {
	gridPoints := make([]*types.Coordinate, len(modelGridPoints))
	for i := range modelGridPoints {
		gridPoints[i] = (*types.Coordinate)(&modelGridPoints[i])
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

func cachedStatesToAPI(modelCachedStates map[string]*model.CellCoverageInfo) map[string]*types.CellCoverageInfo {
	cachedStates := make(map[string]*types.CellCoverageInfo, len(modelCachedStates))

	for key, modelCellCoverageInfo := range modelCachedStates {
		cachedStates[key] = &types.CellCoverageInfo{
			RpCoverageBoundaries: beamCoverageEntriesToAPI(modelCellCoverageInfo.RPCoverageBoundaries),
			CoverageBoundaries:   beamCoverageEntriesToAPI(modelCellCoverageInfo.CoverageBoundaries),
		}
	}
	return cachedStates
}

func beamCoverageEntriesToAPI(modelBeamCoverageEntries map[model.BeamID][]model.CoverageBoundary) []*types.BeamCoverageEntry {
	beamCoverageEntries := make([]*types.BeamCoverageEntry, 0, len(modelBeamCoverageEntries))

	for modelBeamID, coverageBoundaries := range modelBeamCoverageEntries {
		beamID := types.BeamID{Ncgi: modelBeamID.NCGI, CarrierIndex: int32(modelBeamID.CarrierIndex), BeamIndex: int32(modelBeamID.BeamIndex)}
		beamCoverageEntries = append(beamCoverageEntries, &types.BeamCoverageEntry{
			BeamId:             (*types.BeamID)(&beamID),
			CoverageBoundaries: coverageBoundariesToAPI(coverageBoundaries),
		})
	}

	return beamCoverageEntries
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

func CellToModel(cell *types.Cell) *model.Cell {
	return &model.Cell{
		CellConfig: model.CellConfig{
			Carriers: carriersToModel(cell.CellConfig.Carriers),
		},
		NCGI:      cell.NCGI,
		CellType:  cell.CellType,
		Color:     cell.Color,
		MaxUEs:    cell.MaxUEs,
		Neighbors: cell.Neighbors,
		EarfcnDL:  cell.Earfcn,
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
			ShadowingMaps: shadowingMapsToModel(cell.Grid.ShadowingMaps),
			GridPoints:    gridPointsToModel(cell.Grid.GridPointsMaps),
			BoundingBoxes: boundingBoxesToModel(cell.Grid.BoundingBoxes),
		},
		Bwps:         bwpsToModel(cell.Bwps),
		CachedStates: cachedStatesToModel(cell.CachedStates),
	}
}

func carriersToModel(carriers []*types.Carrier) []*model.Carrier {
	modelCarriers := make([]*model.Carrier, len(carriers))
	for i, carrier := range carriers {
		modelCarriers[i] = &model.Carrier{
			Beams:                  beamsToModel(carrier.Beams),
			Center:                 model.Coordinate(*carrier.Center),
			Height:                 carrier.Height,
			ArfcnDL:                carrier.ArfcnDl,
			ArfcnUL:                carrier.ArfcnUl,
			BsChannelBwDL:          carrier.BsChannelBwDl,
			BsChannelBwUL:          carrier.BsChannelBwUl,
			TxPowerDB:              carrier.TxPowerdB,
			VSideLobeAttenuationDB: carrier.VSideLobeAttenuationDb,
			Environment:            carrier.Environment,
			LOS:                    carrier.Los,
		}
	}
	return modelCarriers
}

func beamsToModel(beams []*types.Beam) []*model.Beam {
	modelBeams := make([]*model.Beam, len(beams))
	for i, beam := range beams {
		modelBeams[i] = &model.Beam{
			BeamIndex: int(beam.BeamIndex),
			Azimuth:   beam.Azimuth,
			Tilt:      beam.Tilt,
			H3dBAngle: beam.H3DbAngle,
			V3dBAngle: beam.V3DbAngle,
			MaxGain:   beam.MaxGain,
		}
	}
	return modelBeams
}

func sliceCoordToModel(gridPoints []*types.Coordinate) []model.Coordinate {
	modelGridPoints := make([]model.Coordinate, len(gridPoints))
	for i := range gridPoints {
		modelGridPoints[i] = model.Coordinate(*gridPoints[i])
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

func shadowingMapsToModel(shadowingMaps []*types.ShadowingMapEntry) map[model.BeamID][]float64 {
	modelShadowingMaps := make(map[model.BeamID][]float64, len(shadowingMaps))
	for _, entry := range shadowingMaps {
		beamID := model.BeamID{
			NCGI:         entry.BeamId.Ncgi,
			CarrierIndex: int(entry.BeamId.CarrierIndex),
			BeamIndex:    int(entry.BeamId.BeamIndex),
		}
		modelShadowingMaps[beamID] = entry.ShadowingMap
	}
	return modelShadowingMaps
}

func gridPointsToModel(gridPoints []*types.GridPointsEntry) map[model.BeamID][]model.Coordinate {
	modelGridPoints := make(map[model.BeamID][]model.Coordinate, len(gridPoints))
	for _, entry := range gridPoints {
		beamID := model.BeamID{
			NCGI:         entry.BeamId.Ncgi,
			CarrierIndex: int(entry.BeamId.CarrierIndex),
			BeamIndex:    int(entry.BeamId.BeamIndex),
		}
		modelGridPoints[beamID] = sliceCoordToModel(entry.GridPoints)
	}
	return modelGridPoints
}

func boundingBoxesToModel(boundingBoxes []*types.BoundingBoxEntry) map[model.BeamID]*model.BoundingBox {
	modelBoundingBoxes := make(map[model.BeamID]*model.BoundingBox, len(boundingBoxes))
	for _, entry := range boundingBoxes {
		beamID := model.BeamID{
			NCGI:         entry.BeamId.Ncgi,
			CarrierIndex: int(entry.BeamId.CarrierIndex),
			BeamIndex:    int(entry.BeamId.BeamIndex),
		}
		modelBoundingBoxes[beamID] = (*model.BoundingBox)(entry.BoundingBox)
	}
	return modelBoundingBoxes
}

func cachedStatesToModel(cachedStates map[string]*types.CellCoverageInfo) map[string]*model.CellCoverageInfo {
	modelCachedStates := make(map[string]*model.CellCoverageInfo, len(cachedStates))
	for key, cellCoverageInfo := range cachedStates {
		modelCachedStates[key] = &model.CellCoverageInfo{
			RPCoverageBoundaries: beamCoverageEntriesToModel(cellCoverageInfo.RpCoverageBoundaries),
			CoverageBoundaries:   beamCoverageEntriesToModel(cellCoverageInfo.CoverageBoundaries),
		}
	}
	return modelCachedStates
}

func beamCoverageEntriesToModel(beamCoverageEntries []*types.BeamCoverageEntry) map[model.BeamID][]model.CoverageBoundary {
	modelCoverageBoundaries := make(map[model.BeamID][]model.CoverageBoundary, len(beamCoverageEntries))

	for _, entry := range beamCoverageEntries {
		beamID := model.BeamID{
			NCGI:         entry.BeamId.Ncgi,
			CarrierIndex: int(entry.BeamId.CarrierIndex),
			BeamIndex:    int(entry.BeamId.BeamIndex),
		}
		modelCoverageBoundaries[beamID] = coverageBoundariesToModel(entry.CoverageBoundaries)
	}

	return modelCoverageBoundaries
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
	err := s.cellStore.Add(ctx, CellToModel(request.Cell))
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
	err := s.cellStore.Update(ctx, CellToModel(request.Cell))
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

func (s *Server) ListPatchedCells(request *modelapi.ListPatchedCellsRequest, server modelapi.CellModel_ListPatchedCellsServer) error {
	log.Debugf("Received listing patched cells request: %v", request)

	for index := range s.patchedCells {
		cell := s.patchedCells[index]
		resp := &modelapi.ListCellsResponse{
			Cell: cellToAPI(&cell),
		}
		err := server.Send(resp)
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
