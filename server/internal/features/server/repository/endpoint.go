package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
)

type EndpointRepository struct {
	db      *gorm.DB
	events  *StreamEventPublisher
}

func NewEndpointRepository(db *gorm.DB) *EndpointRepository {
	return &EndpointRepository{db: db}
}

func NewEndpointRepositoryWithDeps(
	db *gorm.DB,
	events *StreamEventPublisher,
) *EndpointRepository {
	return &EndpointRepository{
		db:     db,
		events: events,
	}
}

func RegisterEndpointRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return NewEndpointRepositoryWithDeps(
			dbWrapper.GetDB(),
			do.MustInvoke[*StreamEventPublisher](i),
		), nil
	})
}

func (er *EndpointRepository) GetByIDs(ctx context.Context, ids []uint) ([]domain.Endpoint, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return gorm.G[domain.Endpoint](er.db).Where("id IN ?", ids).Find(ctx)
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

		if err := er.events.Publish(ctx, "deleted", &ep); err != nil {
			return fmt.Errorf("failed to publish delete event: %w", err)
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

		return er.events.Publish(ctx, "created", &endpoint)
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

	for i := range endpoints {
		if err := er.events.Publish(ctx, "created", &endpoints[i]); err != nil {
			return fmt.Errorf("failed to publish create event: %w", err)
		}
	}

	return nil
}
