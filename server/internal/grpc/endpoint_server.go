package grpc

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"

	endpointv1 "github.com/minhnbnt/uptime-monitor-microservices/proto/gen/endpoint/v1"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
)

type EndpointServer struct {
	endpointv1.UnimplementedEndpointServiceServer
	endpointRepo *repository.EndpointRepository
}

func NewEndpointServer(endpointRepo *repository.EndpointRepository) *EndpointServer {
	return &EndpointServer{endpointRepo: endpointRepo}
}

func (s *EndpointServer) GetEndpoints(ctx context.Context, req *endpointv1.GetEndpointsRequest) (*endpointv1.GetEndpointsResponse, error) {
	ids := make([]uint, len(req.EndpointIds))
	for i, id := range req.EndpointIds {
		ids[i] = uint(id)
	}

	endpoints, err := s.endpointRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get endpoints: %w", err)
	}

	resp := &endpointv1.GetEndpointsResponse{
		Endpoints: make([]*endpointv1.EndpointData, 0, len(endpoints)),
	}

	for _, ep := range endpoints {
		resp.Endpoints = append(resp.Endpoints, &endpointv1.EndpointData{
			Id:            uint64(ep.ID),
			ServerId:      uint64(ep.ServerID),
			Url:           ep.URL,
			Method:        ep.Method,
			ExpectedCode:  int32(ep.ExpectedCode),
			IntervalMs:    ep.Interval.Milliseconds(),
			TimeoutMs:     ep.Timeout.Milliseconds(),
			MonitorStatus: string(ep.MonitorStatus),
		})
	}

	return resp, nil
}

func (s *EndpointServer) UpdateMonitorStatus(ctx context.Context, req *endpointv1.UpdateMonitorStatusRequest) (*endpointv1.UpdateMonitorStatusResponse, error) {
	if err := s.endpointRepo.UpdateMonitorStatus(ctx, uint(req.EndpointId), domain.ServerStatus(req.Status)); err != nil {
		return nil, fmt.Errorf("update monitor status: %w", err)
	}

	return &endpointv1.UpdateMonitorStatusResponse{}, nil
}

func StartGRPCServer(ctx context.Context, addr string, endpointRepo *repository.EndpointRepository) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srv := grpc.NewServer()
	endpointv1.RegisterEndpointServiceServer(srv, NewEndpointServer(endpointRepo))

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()

	return srv.Serve(lis)
}
