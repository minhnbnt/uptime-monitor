package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
	monitorrepo "github.com/minhnbnt/uptime-monitor/internal/repository/monitor"
	"github.com/minhnbnt/uptime-monitor/internal/repository/scheduler"
)

type EndpointRepository struct {
	db          *gorm.DB
	scheduler   scheduler.SchedulerRepository
	statusStore *monitorrepo.RedisServerEventRepository
}

func RegisterEndpointRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointRepository, error) {

		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		backend := do.MustInvoke[*scheduler.SchedulerBackend](i)

		var sched scheduler.SchedulerRepository
		if *backend == scheduler.SchedulerBackendTemporal {
			sched = do.MustInvoke[*scheduler.TemporalSchedulerRepository](i)
		} else {
			sched = do.MustInvoke[*scheduler.ZSetScheduleRepository](i)
		}

		return &EndpointRepository{
			db:          dbWrapper.GetDB(),
			scheduler:   sched,
			statusStore: do.MustInvoke[*monitorrepo.RedisServerEventRepository](i),
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

func (er *EndpointRepository) DeleteByServerID(ctx context.Context, serverID uint) error {

	return er.db.Transaction(func(tx *gorm.DB) error {

		ep, err := gorm.G[domain.Endpoint](tx).Where("server_id = ?", serverID).First(ctx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to get endpoint: %w", err)
		}

		if _, err := gorm.G[domain.Endpoint](tx).Where("id = ?", ep.ID).Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete endpoint %d: %w", ep.ID, err)
		}

		if err := er.scheduler.Unregister(ctx, ep.ID); err != nil {
			return fmt.Errorf("failed to unregister endpoint %d: %w", ep.ID, err)
		}

		if err := er.statusStore.DeleteStatus(ctx, ep.ID); err != nil {
			return fmt.Errorf("failed to delete status for endpoint %d: %w", ep.ID, err)
		}

		return nil
	})
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
