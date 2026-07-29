package grpcclient

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

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

		result[uint(ep.Id)] = &domain.Server{
			Model:         gorm.Model{ID: uint(ep.Id)},
			ServerID:      uint(ep.ServerId),
			Namespace:     ep.Namespace,
			Kind:          ep.Kind,
			ObjectID:      ep.ObjectId,
			ContainerName: ep.ContainerName,
			Interval:      time.Duration(ep.IntervalMs) * time.Millisecond,
			Timeout:       time.Duration(ep.TimeoutMs) * time.Millisecond,
		}
	}

	return result, nil
}
