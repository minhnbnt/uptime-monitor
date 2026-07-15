package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type ServerEventRepository struct {
	db *gorm.DB
}

func NewServerEventRepository(db *gorm.DB) *ServerEventRepository {
	return &ServerEventRepository{db: db}
}

func newServerEventRepository(i do.Injector) (*ServerEventRepository, error) {
	dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
	return NewServerEventRepository(dbWrapper.GetDB()), nil
}

func RegisterServerEventRepository(i do.Injector) {
	do.Provide(i, newServerEventRepository)
}

func (r *ServerEventRepository) Save(ctx context.Context, event *domain.ServerEvent) error {
	return gorm.G[domain.ServerEvent](r.db).Create(ctx, event)
}

func (r *ServerEventRepository) GetLatestStatus(ctx context.Context, endpointID uint) (domain.ServerStatus, error) {

	event, err := gorm.G[domain.ServerEvent](r.db).
		Where("endpoint_id = ?", endpointID).
		Order("time DESC").
		First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get latest status: %w", err)
	}

	return event.Status, nil
}
