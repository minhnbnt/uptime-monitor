package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
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

type EnrichedEvent struct {
	EndpointID uint
	Status     domain.ServerStatus
	Time       time.Time
	URL        string
	ServerName string
}

func (r *ServerEventRepository) GetEventsByUserAndDateRange(
	ctx context.Context, userID uint, from, to time.Time,
) ([]domain.ServerEvent, error) {
	return gorm.G[domain.ServerEvent](r.db).Raw(`
		SELECT se.id, se.endpoint_id, se.status, se.time
		FROM server_events se
		JOIN endpoints e ON e.id = se.endpoint_id AND e.deleted_at IS NULL
		JOIN servers s ON s.id = e.server_id AND s.deleted_at IS NULL
		JOIN users u ON u.id = s.created_by_id
		WHERE u.id = ? AND se.time BETWEEN ? AND ?
		ORDER BY se.time DESC
	`, userID, from, to).Find(ctx)
}

func (r *ServerEventRepository) GetEnrichedEventsByUser(
	ctx context.Context, userID uint, from, to time.Time,
) ([]EnrichedEvent, error) {
	return gorm.G[EnrichedEvent](r.db).Raw(`
		SELECT se.endpoint_id, se.status, se.time, e.url, COALESCE(s.name, '') AS server_name
		FROM server_events se
		JOIN endpoints e ON e.id = se.endpoint_id AND e.deleted_at IS NULL
		JOIN servers s ON s.id = e.server_id AND s.deleted_at IS NULL
		JOIN users u ON u.id = s.created_by_id
		WHERE u.id = ? AND se.time BETWEEN ? AND ?
		ORDER BY se.time DESC
	`, userID, from, to).Find(ctx)
}
