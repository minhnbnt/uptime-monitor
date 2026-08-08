package service

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis/cache"
)

type ServerProvider struct {
	client *grpcclient.EndpointClient
	cache  *cache.ServerMetaCache
	logger *slog.Logger
}

func RegisterServerProvider(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerProvider, error) {
		return &ServerProvider{
			client: do.MustInvoke[*grpcclient.EndpointClient](i),
			cache:  do.MustInvoke[*cache.ServerMetaCache](i),
			logger: do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (p *ServerProvider) Get(ctx context.Context, id uint) (*domain.Server, error) {

	results, err := p.GetBatch(ctx, []uint{id})
	if err != nil {
		return nil, err
	}

	result, has := results[id]
	if !has {
		return nil, fmt.Errorf("server not found: %d", id)
	}

	return result, nil
}

func (p *ServerProvider) GetBatch(ctx context.Context, ids []uint) (map[uint]*domain.Server, error) {

	if len(ids) == 0 {
		return make(map[uint]*domain.Server), nil
	}

	servers, err := p.cache.MGet(ctx, ids)
	if err != nil {
		p.logger.Error("failed to get servers from cache", "error", err)
		servers = make(map[uint]*domain.Server)
	}

	missed := lo.Filter(ids, func(id uint, _ int) bool {
		_, has := servers[id]
		return !has
	})

	if len(missed) == 0 {
		return servers, nil
	}

	batch, err := p.client.GetBatch(ctx, missed)
	if err != nil {
		return nil, err
	}

	maps.Copy(servers, batch)

	if err := p.cache.SetMulti(ctx, lo.Values(servers)); err != nil {
		p.logger.Error("failed to set servers in cache", "error", err)
	}

	return servers, nil
}
