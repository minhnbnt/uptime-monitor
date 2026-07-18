package handler

import (
	"context"
	"fmt"
	"net"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"google.golang.org/grpc"

	endpointv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/endpoint/v1"
	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/repository"
)

func RegisterEndpointServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointServer, error) {
		return NewEndpointServer(
			do.MustInvoke[*repository.EndpointRepository](i),
		), nil
	})
}

type EndpointServer struct {
	endpointv1.UnimplementedEndpointServiceServer
	endpointRepo *repository.EndpointRepository
}

func NewEndpointServer(endpointRepo *repository.EndpointRepository) *EndpointServer {
	return &EndpointServer{endpointRepo: endpointRepo}
}

func (s *EndpointServer) GetEndpoints(ctx context.Context, req *endpointv1.GetEndpointsRequest) (*endpointv1.GetEndpointsResponse, error) {

	ids := lo.Map(req.EndpointIds, func(id uint64, _ int) uint {
		return uint(id)
	})

	endpoints, err := s.endpointRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get endpoints: %w", err)
	}

	resp := &endpointv1.GetEndpointsResponse{}
	resp.Endpoints = lo.Map(
		endpoints,
		func(ep domain.Endpoint, _ int) *endpointv1.EndpointData {
			return &endpointv1.EndpointData{
				Id:           uint64(ep.ID),
				ServerId:     uint64(ep.ServerID),
				Url:          ep.URL,
				Method:       ep.Method,
				ExpectedCode: int32(ep.ExpectedCode),
				IntervalMs:   ep.Interval.Milliseconds(),
				TimeoutMs:    ep.Timeout.Milliseconds(),
			}
		},
	)

	return resp, nil
}

func StartGRPCServer(
	ctx context.Context, addr string,
	endpointSrv *EndpointServer,
	serverSrv *ServerServer,
) error {

	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srv := grpc.NewServer()
	endpointv1.RegisterEndpointServiceServer(srv, endpointSrv)
	serverv1.RegisterServerServiceServer(srv, serverSrv)

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()

	return srv.Serve(lis)
}
