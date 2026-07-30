package handler

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	endpointv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/endpoint/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/repository"
)

func RegisterEndpointServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointServer, error) {
		return NewEndpointServer(
			do.MustInvoke[*repository.ServerRepository](i),
			do.MustInvoke[*repository.ServerHttpConfigRepository](i),
		), nil
	})
}

type EndpointServer struct {
	endpointv1.UnimplementedEndpointServiceServer
	serverRepo  *repository.ServerRepository
	httpCfgRepo *repository.ServerHttpConfigRepository
}

func NewEndpointServer(serverRepo *repository.ServerRepository, httpCfgRepo *repository.ServerHttpConfigRepository) *EndpointServer {
	return &EndpointServer{serverRepo: serverRepo, httpCfgRepo: httpCfgRepo}
}

func (s *EndpointServer) GetEndpoints(ctx context.Context, req *endpointv1.GetEndpointsRequest) (*endpointv1.GetEndpointsResponse, error) {

	ids := lo.Map(req.EndpointIds, func(id uint64, _ int) uint {
		return uint(id)
	})

	if len(ids) == 0 {
		return &endpointv1.GetEndpointsResponse{}, nil
	}

	servers, err := s.serverRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	httpCfgs, err := s.httpCfgRepo.GetByServerIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	resp := &endpointv1.GetEndpointsResponse{}
	resp.Endpoints = lo.Map(
		servers,
		func(sv domain.Server, _ int) *endpointv1.EndpointData {

			ed := &endpointv1.EndpointData{
				Id:            uint64(sv.ID),
				ServerId:      uint64(sv.ID),
				Namespace:     sv.Namespace,
				Kind:          sv.Kind,
				ObjectId:      sv.ObjectID,
				ContainerName: sv.ContainerName,
				IntervalMs:    sv.Interval.Milliseconds(),
				TimeoutMs:     sv.Timeout.Milliseconds(),
			}

			ed.PingType = endpointv1.PingType_PING_TYPE_POD_STATUS
			if cfg, ok := httpCfgs[sv.ID]; ok {
				ed.PingType = endpointv1.PingType_PING_TYPE_HTTP_DNS
				ed.HttpDnsConfig = &endpointv1.HttpDnsConfig{
					Port:          int32(cfg.Port),
					EndpointPath:  cfg.EndpointPath,
					ExpectedCode:  int32(cfg.ExpectedCode),
					BodyCheckExpr: cfg.BodyCheckExpr,
				}
			}

			return ed
		},
	)

	return resp, nil
}
