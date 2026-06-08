package scheduler

import (
	"context"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type EndpointFetcher struct {
	db *gorm.DB
}

func RegisterEndpointFetcher(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointFetcher, error) {
		wrapper := do.MustInvoke[*config.GORMWrapper](i)
		return &EndpointFetcher{db: wrapper.GetDB()}, nil
	})
}

func (f *EndpointFetcher) Fetch(ctx context.Context, ids ...uint) ([]domain.Endpoint, error) {

	if len(ids) == 0 {
		return nil, nil
	}

	return gorm.G[domain.Endpoint](f.db).Where("id IN ?", ids).Find(ctx)
}

func (f *EndpointFetcher) GetAll(ctx context.Context, batch int, callback func([]domain.Endpoint, int) error) error {
	return gorm.G[domain.Endpoint](f.db).FindInBatches(ctx, batch, callback)
}
