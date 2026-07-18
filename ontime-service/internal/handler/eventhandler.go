package handler

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"google.golang.org/grpc"

	eventv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/event/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/service"
)

type EventRecorderServer struct {
	eventv1.UnsafeEventRecorderServiceServer
	eventService *service.EventService
}

func RegisterEventRecorderServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EventRecorderServer, error) {
		return &EventRecorderServer{
			eventService: do.MustInvoke[*service.EventService](i),
		}, nil
	})

	gRPCServer := do.MustInvoke[*grpc.Server](i)
	eventv1.RegisterEventRecorderServiceServer(gRPCServer, do.MustInvoke[*EventRecorderServer](i))
}

func (s *EventRecorderServer) RecordEvent(ctx context.Context, req *eventv1.RecordEventRequest) (*eventv1.RecordEventResponse, error) {

	err := s.eventService.RecordEvent(ctx, dto.RecordEventRequest{
		Status:     dto.ServerStatus(req.Status),
		EndpointID: uint(req.EndpointId),
	})

	if err != nil {
		return nil, err
	}

	return &eventv1.RecordEventResponse{}, nil
}

type StatusServer struct {
	eventv1.UnsafeStatusServiceServer
	eventService *service.EventService
}

func RegisterStatusServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*StatusServer, error) {
		return &StatusServer{
			eventService: do.MustInvoke[*service.EventService](i),
		}, nil
	})
}

func (s *StatusServer) GetCurrentStatuses(ctx context.Context, req *eventv1.GetCurrentStatusesRequest) (*eventv1.GetCurrentStatusesResponse, error) {

	if len(req.EndpointIds) == 0 {
		return &eventv1.GetCurrentStatusesResponse{}, nil
	}

	ids := lo.Map(req.EndpointIds, func(id uint64, _ int) uint { return uint(id) })

	statuses, err := s.eventService.GetCurrentStatuses(ctx, ids)
	if err != nil {
		return nil, err
	}

	mapped := lo.Map(statuses, func(st dto.EndpointStatus, _ int) *eventv1.EndpointStatus {
		return &eventv1.EndpointStatus{
			EndpointId: uint64(st.EndpointID),
			Status:     st.Status.String(),
		}
	})

	return &eventv1.GetCurrentStatusesResponse{Statuses: mapped}, nil
}

func (s *StatusServer) CountByStatus(ctx context.Context, req *eventv1.CountByStatusRequest) (*eventv1.CountByStatusResponse, error) {

	ids := lo.Map(req.EndpointIds, func(id uint64, _ int) uint { return uint(id) })

	count, err := s.eventService.CountByStatus(ctx, ids)
	if err != nil {
		return nil, err
	}

	return &eventv1.CountByStatusResponse{
		Online:  count.Online,
		Offline: count.Offline,
	}, nil
}

var (
	_ eventv1.EventRecorderServiceServer = (*EventRecorderServer)(nil)
	_ eventv1.StatusServiceServer         = (*StatusServer)(nil)
)
