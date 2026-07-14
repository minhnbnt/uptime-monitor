package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	monitorrepo "github.com/minhnbnt/uptime-monitor/internal/features/ping/repository"
	"github.com/minhnbnt/uptime-monitor/internal/features/ping/scheduler"
)

type endpointMetaCache interface {
	SetMulti(ctx context.Context, endpoints []domain.Endpoint) error
	Delete(ctx context.Context, id uint) error
}

type EndpointRepository struct {
	db          *gorm.DB
	scheduler   scheduler.SchedulerRepository
	statusStore *monitorrepo.RedisServerEventRepository
	metaCache   endpointMetaCache
}

func NewEndpointRepository(db *gorm.DB) *EndpointRepository {
	return &EndpointRepository{db: db}
}

func NewEndpointRepositoryWithDeps(
	db *gorm.DB,
	scheduler scheduler.SchedulerRepository,
	statusStore *monitorrepo.RedisServerEventRepository,
	metaCache *scheduler.EndpointMetaCache,
) *EndpointRepository {
	return &EndpointRepository{
		db:          db,
		scheduler:   scheduler,
		statusStore: statusStore,
		metaCache:   metaCache,
	}
}

func getSchedulerRepository(i do.Injector) scheduler.SchedulerRepository {

	cfg := do.MustInvoke[*config.Config](i)
	log := do.MustInvoke[*slog.Logger](i)

	backend := cfg.Scheduler.Backend

	switch backend {
	case "temporal":
		return do.MustInvoke[*scheduler.TemporalSchedulerRepository](i)

	case "redis":
		return do.MustInvoke[*scheduler.ZSetScheduleRepository](i)

	default:
		log.Error(
			"unsupported scheduler backend",
			slog.String("backend", backend),
		)
		panic("unsupported scheduler backend")
	}
}

func RegisterEndpointRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointRepository, error) {

		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)

		return NewEndpointRepositoryWithDeps(
			dbWrapper.GetDB(),
			getSchedulerRepository(i),
			do.MustInvoke[*monitorrepo.RedisServerEventRepository](i),
			do.MustInvoke[*scheduler.EndpointMetaCache](i),
		), nil
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

		if err := er.metaCache.Delete(ctx, ep.ID); err != nil {
			return fmt.Errorf("failed to delete meta cache for endpoint %d: %w", ep.ID, err)
		}

		return nil
	})
}

func (er *EndpointRepository) UpsertEndpoint(ctx context.Context, endpoint domain.Endpoint) error {

	return er.db.Transaction(func(tx *gorm.DB) error {

		queryClause := clause.OnConflict{
			Columns: []clause.Column{{Name: "server_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"url", "method",
				"expected_code",
				"interval",
				"timeout",
			}),
		}

		if err := gorm.G[domain.Endpoint](tx, queryClause).Create(ctx, &endpoint); err != nil {
			return err
		}

		if err := er.scheduler.Register(ctx, &endpoint); err != nil {
			return err
		}

		return er.metaCache.Delete(ctx, endpoint.ID)
	})
}

func (er *EndpointRepository) UpdateMonitorStatus(ctx context.Context, endpointID uint, status domain.ServerStatus) error {

	affected, err := gorm.G[domain.Endpoint](er.db).
		Where("id = ?", endpointID).
		Update(ctx, "monitor_status", status)

	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("endpoint %d: %w", endpointID, apperrors.ErrNotFound)
	}

	return nil
}

func (er *EndpointRepository) BatchCreateEndpoints(ctx context.Context, endpoints []domain.Endpoint) error {

	result := er.db.WithContext(ctx).Create(endpoints)

	if err := result.Error; err != nil {
		return fmt.Errorf("failed to batch create endpoints: %w", err)
	}

	if err := er.scheduler.RegisterBatch(ctx, endpoints); err != nil {
		return fmt.Errorf("failed to batch register endpoints: %w", err)
	}

	if err := er.metaCache.SetMulti(ctx, endpoints); err != nil {
		return fmt.Errorf("failed to set meta cache: %w", err)
	}

	return nil
}
