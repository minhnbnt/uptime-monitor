package grpcclient

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"

	endpointv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/endpoint/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type EndpointClient struct {
	client endpointv1.EndpointServiceClient
}

func RegisterEndpointClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointClient, error) {

		conn := do.MustInvoke[*config.GRPCClientWrapper](i)
		client := endpointv1.NewEndpointServiceClient(conn.GetConn())

		return &EndpointClient{client: client}, nil
	})
}

func (c *EndpointClient) GetBatch(ctx context.Context, ids []uint) (map[uint]*domain.Server, error) {

	endpointIDs := make([]uint64, len(ids))
	for i, id := range ids {
		endpointIDs[i] = uint64(id)
	}

	resp, err := c.client.GetEndpoints(ctx, &endpointv1.GetEndpointsRequest{
		EndpointIds: endpointIDs,
	})

	if err != nil {
		return nil, fmt.Errorf("get endpoints: %w", err)
	}

	result := make(map[uint]*domain.Server, len(resp.Endpoints))
	for _, ep := range resp.Endpoints {

		sv := &domain.Server{
			ID:            uint(ep.Id),
			Namespace:     ep.Namespace,
			Kind:          ep.Kind,
			ObjectID:      ep.ObjectId,
			ContainerName: ep.ContainerName,
			Interval:      time.Duration(ep.IntervalMs) * time.Millisecond,
			Timeout:       time.Duration(ep.TimeoutMs) * time.Millisecond,
		}

		if cfg := ep.GetHttpDnsConfig(); cfg != nil {
			sv.HTTPConfig = &domain.ServerHTTPConfig{
				Port:          int(cfg.Port),
				EndpointPath:  cfg.EndpointPath,
				ExpectedCode:  int(cfg.ExpectedCode),
				BodyCheckExpr: cfg.BodyCheckExpr,
				Method:        cfg.Method,
			}
		}

		result[uint(ep.Id)] = sv
	}

	return result, nil
}
