package repository

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
)

type EndpointRepository struct {
	db *gorm.DB
}

func NewEndpointRepository(db *gorm.DB) *EndpointRepository {
	return &EndpointRepository{db: db}
}

func RegisterEndpointRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return NewEndpointRepository(dbWrapper.GetDB()), nil
	})
}

func (er *EndpointRepository) GetByIDs(
	ctx context.Context, ids []uint,
) ([]domain.Endpoint, error) {

	if len(ids) == 0 {
		return nil, nil
	}

	return gorm.G[domain.Endpoint](er.db).
		Where("id IN ?", ids).
		Find(ctx)
}

func (er *EndpointRepository) GetByServerID(
	ctx context.Context, serverID uint,
) (*domain.Endpoint, error) {

	endpoint, err := gorm.G[domain.Endpoint](er.db).
		Where("server_id = ?", serverID).
		First(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}

	return &endpoint, nil
}

func (er *EndpointRepository) DeleteByServerID(ctx context.Context, serverID uint) error {

	_, err := gorm.G[domain.Endpoint](er.db).
		Where("server_id = ?", serverID).
		Delete(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete endpoint: %w", err)
	}

	return nil
}

func (er *EndpointRepository) UpsertEndpoint(
	ctx context.Context,
	endpoint domain.Endpoint,
) error {

	queryClause := clause.OnConflict{
		Columns: []clause.Column{{Name: "server_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"url", "method",
			"expected_code",
			"interval",
			"timeout",
		}),
	}

	return gorm.G[domain.Endpoint](er.db, queryClause).
		Create(ctx, &endpoint)
}

func (er *EndpointRepository) BatchCreateEndpoints(
	ctx context.Context,
	endpoints []domain.Endpoint,
) error {

	result := er.db.WithContext(ctx).Create(endpoints)

	if err := result.Error; err != nil {
		return fmt.Errorf("failed to batch create endpoints: %w", err)
	}

	return nil
}
