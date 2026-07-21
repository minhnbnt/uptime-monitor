package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcclient"
)

type EndpointProvider struct {
	client *grpcclient.EndpointClient
	cache  *EndpointMetaCache
	logger *slog.Logger
}

func RegisterEndpointProvider(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointProvider, error) {
		return &EndpointProvider{
			client: do.MustInvoke[*grpcclient.EndpointClient](i),
			cache:  do.MustInvoke[*EndpointMetaCache](i),
			logger: do.MustInvoke[*slog.Logger](i),
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

	endpoints, err := p.cache.MGet(ctx, ids)
	if err != nil {
		p.logger.Error("failed to get endpoints from cache", "error", err)
		endpoints = make(map[uint]*domain.Endpoint)
	}

	missed := lo.Filter(ids, func(id uint, _ int) bool {
		_, has := endpoints[id]
		return !has
	})

	if len(missed) == 0 {
		return endpoints, nil
	}

	batch, err := p.client.GetBatch(ctx, missed)
	if err != nil {
		return nil, err
	}

	maps.Copy(endpoints, batch)

	if err := p.cache.SetMulti(ctx, lo.Values(endpoints)); err != nil {
		p.logger.Error("failed to set endpoints in cache", "error", err)
	}

	return endpoints, nil
}
