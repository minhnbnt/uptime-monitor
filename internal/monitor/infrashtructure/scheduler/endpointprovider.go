package scheduler

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"

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

	ep, err := p.cache.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if ep != nil {
		return ep, nil
	}

	eps, err := p.fetcher.Fetch(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(eps) == 0 {
		return nil, nil
	}

	if err := p.cache.Set(ctx, &eps[0]); err != nil {
		return nil, fmt.Errorf("cache set %d: %w", id, err)
	}

	return &eps[0], nil
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

	var missIDs []uint
	for _, id := range ids {
		if ep, ok := cached[id]; ok {
			result[id] = ep
		} else {
			missIDs = append(missIDs, id)
		}
	}

	if len(missIDs) == 0 {
		return result, nil
	}

	eps, err := p.fetcher.Fetch(ctx, missIDs...)
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
