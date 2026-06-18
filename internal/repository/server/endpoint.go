package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
	monitorrepo "github.com/minhnbnt/uptime-monitor/internal/features/ping/repository"
	"github.com/minhnbnt/uptime-monitor/internal/features/ping/scheduler"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

type EndpointRepository struct {
	db          *gorm.DB
	scheduler   scheduler.SchedulerRepository
	statusStore *monitorrepo.RedisServerEventRepository
}

func NewEndpointRepository(db *gorm.DB) *EndpointRepository {
	return &EndpointRepository{db: db}
}

func getSchedulerRepository(i do.Injector) scheduler.SchedulerRepository {

	cfg := do.MustInvoke[*config.Config](i)
	log := do.MustInvoke[logger.Logger](i)

	backend := cfg.Scheduler.Backend

	switch backend {
	case "temporal":
		return do.MustInvoke[*scheduler.TemporalSchedulerRepository](i)

	case "redis":
		return do.MustInvoke[*scheduler.ZSetScheduleRepository](i)

	default:
		log.Panic(
			"unsupported scheduler backend",
			logger.String("backend", backend),
		)
	}

	return nil
}

func RegisterEndpointRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointRepository, error) {

		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)

		return &EndpointRepository{
			db:          dbWrapper.GetDB(),
			scheduler:   getSchedulerRepository(i),
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

		txWithClauses := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "server_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"url", "method",
				"expected_code",
				"interval",
				"timeout",
				"status",
			}),
		})

		if err := gorm.G[domain.Endpoint](txWithClauses).Create(ctx, &endpoint); err != nil {
			return err
		}

		return er.scheduler.Register(ctx, &endpoint)
	})
}

func (er *EndpointRepository) BatchCreateEndpoints(ctx context.Context, endpoints []domain.Endpoint) error {

	result := er.db.WithContext(ctx).Create(endpoints)

	if err := result.Error; err != nil {
		return fmt.Errorf("failed to batch create endpoints: %w", err)
	}

	return nil
}
