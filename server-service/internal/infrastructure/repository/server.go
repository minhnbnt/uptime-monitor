package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/grpcclient"
)

type ServerRepository struct {
	db          *gorm.DB
	eventClient grpcclient.StatusClient
}

func NewServerRepository(
	db *gorm.DB,
	eventClient grpcclient.StatusClient,
) *ServerRepository {
	return &ServerRepository{db: db, eventClient: eventClient}
}

func RegisterServerRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerRepository, error) {

		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		eventClient := do.MustInvoke[grpcclient.StatusClient](i)

		return &ServerRepository{
			db:          dbWrapper.GetDB(),
			eventClient: eventClient,
		}, nil
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
		Preload("Endpoint", nil).
		Limit(limit).
		Offset(offset).
		Find(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get servers: %w", err)
	}

	return servers, nil
}

func (sr *ServerRepository) Create(ctx context.Context, s *domain.Server) error {
	return gorm.G[domain.Server](sr.db).Create(ctx, s)
}

func (sr *ServerRepository) GetByID(
	ctx context.Context, id uint,
) (*domain.Server, error) {

	server, err := gorm.G[domain.Server](sr.db).
		Preload("Endpoint", nil).
		Where("id = ?", id).
		First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("server %d: %w", id, apperrors.ErrNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return &server, nil
}

func (sr *ServerRepository) Update(ctx context.Context, s *domain.Server) error {

	rowAffected, err := gorm.G[domain.Server](sr.db).
		Where("id = ?", s.ID).
		Updates(ctx, *s)

	if err != nil {
		return err
	}

	if rowAffected == 0 {
		return fmt.Errorf("server %d: %w", s.ID, apperrors.ErrNotFound)
	}

	return nil
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

	endpointIDs := []uint{}
	result := sr.db.WithContext(ctx).
		Table("endpoints e").
		Joins("JOIN servers s ON s.id = e.server_id").
		Where("s.created_by_id = ?", createdByID).
		Pluck("e.id", &endpointIDs)

	if err := result.Error; err != nil {
		return 0, 0, 0, fmt.Errorf("get endpoint ids: %w", err)
	}

	total = int64(len(endpointIDs))
	if total == 0 {
		return 0, 0, 0, nil
	}

	online, offline, err = sr.eventClient.CountByStatus(ctx, endpointIDs)
	if err != nil {
		return 0, 0, 0, err
	}

	return total, online, offline, nil
}

func (sr *ServerRepository) BatchCreateServers(
	ctx context.Context,
	servers []domain.Server,
) error {

	result := sr.db.WithContext(ctx).Create(&servers)

	if err := result.Error; err != nil {
		return fmt.Errorf("failed to batch create servers: %w", err)
	}

	return nil
}
