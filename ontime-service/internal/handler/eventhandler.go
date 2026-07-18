package handler

import (
	"context"

	"github.com/samber/do/v2"

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

var _ eventv1.EventRecorderServiceServer = (*EventRecorderServer)(nil)
