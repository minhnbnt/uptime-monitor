package repository

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/grpcclient"
)

type ServerRepository struct {
	db          *gorm.DB
	eventClient grpcclient.StatusClient
}

func NewServerRepository(db *gorm.DB, eventClient grpcclient.StatusClient) *ServerRepository {
	return &ServerRepository{db: db, eventClient: eventClient}
}

func RegisterServerRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerRepository, error) {

		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		eventClient := do.MustInvoke[grpcclient.StatusClient](i)

		return NewServerRepository(
			dbWrapper.GetDB(),
			eventClient,
		), nil
	})
}

func (sr *ServerRepository) Count(ctx context.Context, createdByID uint) (int64, error) {
	return gorm.G[domain.Server](sr.db).
		Where("created_by_id = ?", createdByID).
		Count(ctx, "id")
}

func (sr *ServerRepository) List(
	ctx context.Context,
	createdByID uint,
	limit, offset int,
) ([]domain.Server, error) {

	servers, err := gorm.G[domain.Server](sr.db).
		Where("created_by_id = ?", createdByID).
		Limit(limit).
		Offset(offset).
		Find(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get servers: %w", err)
	}

	return servers, nil
}

func (sr *ServerRepository) Create(ctx context.Context, s *domain.Server, config *domain.ServerHttpConfig) error {
	return sr.db.Transaction(func(tx *gorm.DB) error {

		if err := gorm.G[domain.Server](tx).Create(ctx, s); err != nil {
			return err
		}

		if config == nil {
			return nil
		}

		config.ServerID = s.ID
		return gorm.G[domain.ServerHttpConfig](tx).Create(ctx, config)
	})
}

func (sr *ServerRepository) GetByIDs(ctx context.Context, ids []uint) ([]domain.Server, error) {

	if len(ids) == 0 {
		return nil, nil
	}

	servers, err := gorm.G[domain.Server](sr.db).
		Where("id IN ?", ids).
		Find(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get servers by ids: %w", err)
	}

	return servers, nil
}

func (sr *ServerRepository) GetByID(ctx context.Context, id uint) (*domain.Server, error) {

	results, err := sr.GetByIDs(ctx, []uint{id})
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("server %d: %w", id, apperrors.ErrNotFound)
	}

	return &results[0], nil
}

func (sr *ServerRepository) Update(ctx context.Context, s *domain.Server, config *domain.ServerHttpConfig) error {
	return sr.db.Transaction(func(tx *gorm.DB) error {

		rowAffected, err := gorm.G[domain.Server](tx).
			Where("id = ?", s.ID).
			Updates(ctx, *s)

		if err != nil {
			return err
		}

		if rowAffected == 0 {
			return fmt.Errorf("server %d: %w", s.ID, apperrors.ErrNotFound)
		}

		if config != nil {

			clauses := clause.OnConflict{
				Columns:   []clause.Column{{Name: "server_id"}},
				UpdateAll: true,
			}

			config.ServerID = s.ID
			result := tx.WithContext(ctx).Clauses(clauses).Create(config)

			return result.Error
		}

		_, err = gorm.G[domain.ServerHttpConfig](tx).
			Where("server_id = ?", s.ID).
			Delete(ctx)

		return err
	})
}

func (sr *ServerRepository) Delete(ctx context.Context, id uint) error {

	rowAffected, err := gorm.G[domain.Server](sr.db).Where("id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}

	if rowAffected == 0 {
		return fmt.Errorf("server %d: %w", id, apperrors.ErrNotFound)
	}

	return nil
}

func (sr *ServerRepository) CountByStatus(
	ctx context.Context, createdByID uint,
) (total, online, offline int64, err error) {

	total, err = sr.Count(ctx, createdByID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("count servers: %w", err)
	}

	if total == 0 {
		return 0, 0, 0, nil
	}

	online, offline, err = sr.eventClient.CountByStatus(ctx, createdByID)
	if err != nil {
		return 0, 0, 0, err
	}

	return total, online, offline, nil
}

func (sr *ServerRepository) BatchCreateServers(
	ctx context.Context, servers []domain.Server,
) error {

	result := sr.db.WithContext(ctx).Create(&servers)

	if err := result.Error; err != nil {
		return fmt.Errorf("failed to batch create servers: %w", err)
	}

	return nil
}

func (sr *ServerRepository) ExistsByNamespaceObjectID(ctx context.Context, namespace, objectID string) (bool, error) {
	count, err := gorm.G[domain.Server](sr.db).
		Where("namespace = ? AND object_id = ?", namespace, objectID).
		Count(ctx, "id")
	if err != nil {
		return false, fmt.Errorf("check server existence: %w", err)
	}
	return count > 0, nil
}
