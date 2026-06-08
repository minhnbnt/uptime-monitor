package repository

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type EndpointRepository struct {
	db        *gorm.DB
	scheduler SchedulerRepository
}

func RegisterEndpointRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointRepository, error) {

		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		backend := do.MustInvoke[*SchedulerBackend](i)

		var scheduler SchedulerRepository
		if *backend == SchedulerBackendTemporal {
			scheduler = do.MustInvoke[*PingSchedulerRepository](i)
		} else {
			scheduler = do.MustInvoke[*ZSetSchedulerRepository](i)
		}

		return &EndpointRepository{
			db:        dbWrapper.GetDB(),
			scheduler: scheduler,
		}, nil
	})
}

func (er *EndpointRepository) GetByServerID(ctx context.Context, serverID uint) (*domain.Endpoint, error) {

	endpoint, err := gorm.G[domain.Endpoint](er.db).Where("server_id = ?", serverID).First(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}

	return &endpoint, nil
}

func (er *EndpointRepository) UpsertEndpoint(ctx context.Context, endpoint domain.Endpoint) error {

	return er.db.Transaction(func(tx *gorm.DB) error {

		err := gorm.G[domain.Endpoint](tx).Create(ctx, &endpoint)
		if err != nil {
			return err
		}

		return er.scheduler.Register(ctx, &endpoint)
	})
}
