// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package ues

import (
	"context"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/store/event"
	"github.com/nfvri/ran-simulator/pkg/store/ues"

	modelapi "github.com/nfvri/onos-api/go/onos/ransim/model"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"
	liblog "github.com/onosproject/onos-lib-go/pkg/logging"
	service "github.com/onosproject/onos-lib-go/pkg/northbound"
	"google.golang.org/grpc"
)

var log = liblog.GetLogger()

// NewService returns a new model Service
func NewService(ueStore ues.Store, patchedUEs []model.UE) service.Service {
	return &Service{
		ueStore:    ueStore,
		patchedUEs: patchedUEs,
	}
}

// Service is a Service implementation for administration.
type Service struct {
	service.Service
	ueStore    ues.Store
	patchedUEs []model.UE
}

// Register registers the TrafficSim Service with the gRPC server.
func (s *Service) Register(r *grpc.Server) {
	server := &Server{
		ueStore:    s.ueStore,
		patchedUEs: s.patchedUEs,
	}
	modelapi.RegisterUEModelServer(r, server)
}

// Server implements the Routes gRPC service for administrative facilities.
type Server struct {
	ueStore    ues.Store
	patchedUEs []model.UE
}

// GetUECount gets the number of UEs
func (s *Server) GetUECount(ctx context.Context, request *modelapi.GetUECountRequest) (*modelapi.GetUECountResponse, error) {
	return &modelapi.GetUECountResponse{Count: uint32(s.ueStore.Len(ctx))}, nil
}

// SetUECount sets the number of UEs
func (s *Server) SetUECount(ctx context.Context, request *modelapi.SetUECountRequest) (*modelapi.SetUECountResponse, error) {
	s.ueStore.SetUECount(ctx, uint(request.Count))
	return &modelapi.SetUECountResponse{}, nil
}

func UEToAPI(modelUE *model.UE) *types.Ue {
	return &types.Ue{
		IMSI:          modelUE.IMSI,
		Type:          string(modelUE.Type),
		Location:      (*types.Coordinate)(&modelUE.Location),
		Heading:       modelUE.Heading,
		CRNTI:         modelUE.CRNTI,
		Height:        modelUE.Height,
		IsAdmitted:    modelUE.IsAdmitted,
		RrcState:      uint32(modelUE.RrcState),
		FiveQi:        int32(modelUE.FiveQi),
		ServingCells:  ueCellsToAPI([]*model.UECell{modelUE.Cell}),
		NeighborCells: ueCellsToAPI(modelUE.Cells),
	}
}

func ueCellsToAPI(modelUeCells []*model.UECell) []*types.UECell {
	ueCells := make([]*types.UECell, len(modelUeCells))

	for key, ueCell := range modelUeCells {
		beamID := &types.BeamID{Ncgi: ueCell.NCGI, CarrierIndex: int32(ueCell.BeamID.CarrierIndex), BeamIndex: int32(ueCell.BeamID.BeamIndex)}
		ueCells[key] = &types.UECell{
			Ncgi:        ueCell.NCGI,
			BeamId:      beamID,
			Rsrp:        ueCell.Rsrp,
			Rsrq:        ueCell.Rsrq,
			Sinr:        ueCell.Sinr,
			BwpRefs:     bwpsToAPI(ueCell.BwpRefs),
			AvailPrbsDl: uint32(ueCell.AvailPrbsDl),
		}
	}
	return ueCells
}

func bwpsToAPI(modelBWPs []*model.Bwp) []*types.Bwp {
	bwps := make([]*types.Bwp, len(modelBWPs))
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

func UEToModel(ue *types.Ue) *model.UE {
	return &model.UE{
		IMSI:       ue.IMSI,
		Type:       model.UEType(ue.Type),
		Location:   model.Coordinate(*ue.Location),
		Heading:    ue.Heading,
		CRNTI:      ue.CRNTI,
		Height:     ue.Height,
		IsAdmitted: ue.IsAdmitted,
		RrcState:   e2sm_mho.Rrcstatus(ue.RrcState),
		FiveQi:     int(ue.FiveQi),
		Cell:       ueCellsToModel([]*types.UECell{ue.ServingCells[0]})[0],
		Cells:      ueCellsToModel(ue.NeighborCells),
	}
}

func ueCellsToModel(apiUeCells []*types.UECell) []*model.UECell {
	ueCells := make([]*model.UECell, len(apiUeCells))

	for key, ueCell := range apiUeCells {
		ueCells[key] = &model.UECell{
			ID:          ueCell.GnbID,
			NCGI:        ueCell.Ncgi,
			Rsrp:        ueCell.Rsrp,
			Rsrq:        ueCell.Rsrq,
			Sinr:        ueCell.Sinr,
			BwpRefs:     bwpsToModel(ueCell.BwpRefs),
			AvailPrbsDl: int(ueCell.AvailPrbsDl),
		}
	}
	return ueCells
}

func bwpsToModel(apiBWPs []*types.Bwp) []*model.Bwp {
	bwps := make([]*model.Bwp, len(apiBWPs))
	for key, bwp := range apiBWPs {
		bwps[key] = &model.Bwp{
			ID:          bwp.Id,
			Scs:         int(bwp.Scs),
			NumberOfRBs: int(bwp.NumberOfRbs),
			Downlink:    bwp.Downlink,
		}
	}
	return bwps
}

// GetUE returns information on the specified UE
func (s *Server) GetUE(ctx context.Context, request *modelapi.GetUERequest) (*modelapi.GetUEResponse, error) {
	log.Debugf("Received get UE request: %+v", request)
	ue, err := s.ueStore.Get(ctx, request.IMSI)
	if err != nil {
		return nil, err
	}
	return &modelapi.GetUEResponse{Ue: UEToAPI(ue)}, nil
}

// MoveToCell moves the specified UE to the given cell
func (s *Server) MoveToCell(ctx context.Context, request *modelapi.MoveToCellRequest) (*modelapi.MoveToCellResponse, error) {
	log.Infof("Received MoveToCell request: %+v", request)
	err := s.ueStore.MoveToCell(ctx, request.IMSI, request.NCGI, 0)
	if err != nil {
		return nil, err
	}
	return &modelapi.MoveToCellResponse{}, nil
}

// MoveToLocation moves the specified UE to the given location
func (s *Server) MoveToLocation(ctx context.Context, request *modelapi.MoveToLocationRequest) (*modelapi.MoveToLocationResponse, error) {
	log.Debugf("Received MoveToLocation request: %+v", request)
	return &modelapi.MoveToLocationResponse{}, s.ueStore.MoveToCoordinate(ctx, request.IMSI, model.Coordinate(*request.Location), request.Heading)
}

// DeleteUE removes the specified UE
func (s *Server) DeleteUE(ctx context.Context, request *modelapi.DeleteUERequest) (*modelapi.DeleteUEResponse, error) {
	log.Debugf("Received Delete request: %+v", request)
	_, err := s.ueStore.Delete(ctx, request.IMSI)
	return &modelapi.DeleteUEResponse{}, err
}

func eventType(ueEvent ues.UeEvent) modelapi.EventType {
	if ueEvent == ues.Created {
		return modelapi.EventType_CREATED
	} else if ueEvent == ues.Updated {
		return modelapi.EventType_UPDATED
	} else if ueEvent == ues.Deleted {
		return modelapi.EventType_DELETED
	} else {
		return modelapi.EventType_NONE
	}
}

// WatchUEs returns events pertaining to changes in the UE state.
func (s *Server) WatchUEs(request *modelapi.WatchUEsRequest, server modelapi.UEModel_WatchUEsServer) error {
	log.Debugf("Received WatchUEs request: %+v", request)
	ch := make(chan event.Event)
	err := s.ueStore.Watch(server.Context(), ch, ues.WatchOptions{Replay: !request.NoReplay, Monitor: !request.NoSubscribe})

	if err != nil {
		return err
	}

	for ueEvent := range ch {
		response := &modelapi.WatchUEsResponse{
			Ue:   UEToAPI(ueEvent.Value.(*model.UE)),
			Type: eventType(ueEvent.Type.(ues.UeEvent)),
		}
		err := server.Send(response)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListUEs returns list of simulated UEs.
func (s *Server) ListUEs(request *modelapi.ListUEsRequest, server modelapi.UEModel_ListUEsServer) error {
	log.Debugf("Received listing UEs request: %v", request)
	ueList := s.ueStore.ListAllUEs(server.Context())

	for index := range ueList {
		ue := *ueList[index]
		resp := &modelapi.ListUEsResponse{
			Ue: UEToAPI(&ue),
		}
		err := server.Send(resp)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

func (s *Server) ListPatchedUEs(request *modelapi.ListPatchedUEsRequest, server modelapi.UEModel_ListPatchedUEsServer) error {
	log.Debugf("Received listing patched UEs request: %v", request)

	for index := range s.patchedUEs {
		ue := s.patchedUEs[index]
		resp := &modelapi.ListUEsResponse{
			Ue: UEToAPI(&ue),
		}
		err := server.Send(resp)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}
