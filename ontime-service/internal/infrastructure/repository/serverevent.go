package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
)

type ServerEventRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewServerEventRepository(db *gorm.DB, logger *slog.Logger) *ServerEventRepository {
	return &ServerEventRepository{db: db, logger: logger}
}

func newServerEventRepository(i do.Injector) (*ServerEventRepository, error) {
	dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
	logger := do.MustInvoke[*slog.Logger](i)
	return NewServerEventRepository(dbWrapper.GetDB(), logger), nil
}

func RegisterServerEventRepository(i do.Injector) {
	do.Provide(i, newServerEventRepository)
}

func (r *ServerEventRepository) shouldInsert(ctx context.Context, tx *gorm.DB, event *domain.ServerEvent) (bool, error) {

	lowerboundEvent, err := gorm.G[domain.ServerEvent](tx).
		Where("endpoint_id = ?", event.EndpointID).
		Where("time <= ?", event.Time).
		Order("time DESC").
		Order("id DESC").
		First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil
	}

	if err != nil {
		return true, fmt.Errorf("failed to get lowerbound status: %w", err)
	}

	return lowerboundEvent.Status != event.Status, nil
}

// collapseUpperBound removes the immediate successor when it shares the new
// event's status, so the successor is folded into the new event instead of
// leaving a redundant transition on the timeline.
func (r *ServerEventRepository) collapseUpperBound(ctx context.Context, tx *gorm.DB, event *domain.ServerEvent) error {

	upperBoundEvent, err := gorm.G[domain.ServerEvent](tx).
		Where("endpoint_id = ?", event.EndpointID).
		Where("time > ?", event.Time).
		Order("time ASC").
		Order("id DESC").
		First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to get upperbound status: %w", err)
	}

	if upperBoundEvent.Status != event.Status {
		return nil
	}

	_, err = gorm.G[domain.ServerEvent](tx).
		Where("id = ?", upperBoundEvent.ID).
		Where("status = ?", event.Status).
		Delete(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete upperbound event: %w", err)
	}

	return nil
}

func (r *ServerEventRepository) Save(ctx context.Context, event *domain.ServerEvent) error {
	return r.db.Transaction(func(tx *gorm.DB) error {

		shouldInsert, err := r.shouldInsert(ctx, tx, event)
		if err != nil {
			r.logger.Error("failed to check should insert", "error", err)
		}

		if !shouldInsert {
			return nil
		}

		if err := r.collapseUpperBound(ctx, tx, event); err != nil {
			r.logger.Error("failed to collapse upperbound event, continuing with insert", "error", err)
		}

		return gorm.G[domain.ServerEvent](tx).Create(ctx, event)
	})
}
