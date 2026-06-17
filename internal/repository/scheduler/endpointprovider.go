package scheduler

import (
	"context"
	"fmt"
	"maps"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type EndpointProvider struct {
	cache   *EndpointMetaCache
	fetcher *EndpointFetcher
}

func RegisterEndpointProvider(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointProvider, error) {
		return &EndpointProvider{
			cache:   do.MustInvoke[*EndpointMetaCache](i),
			fetcher: do.MustInvoke[*EndpointFetcher](i),
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

	result := make(map[uint]*domain.Endpoint, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	cached, err := p.cache.MGet(ctx, ids)
	if err != nil {
		return nil, err
	}

	maps.Copy(result, cached)
	missedIDs := lo.Filter(ids, func(id uint, _ int) bool {
		_, hit := cached[id]
		return !hit
	})

	if len(missedIDs) == 0 {
		return result, nil
	}

	eps, err := p.fetcher.Fetch(ctx, missedIDs...)
	if err != nil {
		return nil, err
	}

	for i := range eps {
		result[eps[i].ID] = &eps[i]
	}

	if err := p.cache.SetMulti(ctx, eps); err != nil {
		return nil, fmt.Errorf("cache set multi: %w", err)
	}

	return result, nil
}
