package grpc

import (
	"context"

	eventv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/event/v1"
	"github.com/samber/do/v2"
	"google.golang.org/grpc"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/recorder"
)

type EventServer struct {
	eventv1.UnimplementedEventServiceServer
	recorder *recorder.DedupRecorder
}

func RegisterEventServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EventServer, error) {
		return &EventServer{
			recorder: do.MustInvoke[*recorder.DedupRecorder](i),
		}, nil
	})

	gRPCServer := do.MustInvoke[*grpc.Server](i)
	eventv1.RegisterEventServiceServer(gRPCServer, do.MustInvoke[*EventServer](i))
}

func (s *EventServer) RecordEvent(ctx context.Context, req *eventv1.RecordEventRequest) (*eventv1.RecordEventResponse, error) {
	if err := s.recorder.RecordEvent(ctx, uint(req.EndpointId), domain.ServerStatus(req.Status)); err != nil {
		return nil, err
	}
	return &eventv1.RecordEventResponse{}, nil
}
