package scheduler

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcclient"
)

type EndpointProvider struct {
	client *grpcclient.EndpointClient
}

func RegisterEndpointProvider(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointProvider, error) {
		return &EndpointProvider{
			client: do.MustInvoke[*grpcclient.EndpointClient](i),
		}, nil
	})
}

func (p *EndpointProvider) Get(ctx context.Context, id uint) (*domain.Endpoint, error) {
	results, err := p.GetBatch(ctx, []uint{id})
	if err != nil {
		return nil, err
	}

	result, has := results[id]
	if !has {
		return nil, fmt.Errorf("endpoint not found: %d", id)
	}

	return result, nil
}

func (p *EndpointProvider) GetBatch(ctx context.Context, ids []uint) (map[uint]*domain.Endpoint, error) {
	if len(ids) == 0 {
		return make(map[uint]*domain.Endpoint), nil
	}

	return p.client.GetBatch(ctx, ids)
}
