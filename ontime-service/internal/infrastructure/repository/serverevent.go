package repository

import (
	"context"
	"errors"
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

func (r *ServerEventRepository) Save(ctx context.Context, event *domain.ServerEvent) error {
	return r.db.Transaction(func(tx *gorm.DB) error {

		latestEvent, err := gorm.G[domain.ServerEvent](tx).
			Where("endpoint_id = ?", event.EndpointID).
			Order("time DESC").
			First(ctx)

		if errors.Is(err, gorm.ErrRecordNotFound) {
			latestEvent.Status = domain.ServerStatus("unknown")
		}

		if err != nil {

			if !errors.Is(err, gorm.ErrRecordNotFound) {
				r.logger.Error("failed to get latest status", slog.Any("err", err))
			}

			// ignore error, continue with saving the event
			// calculator can handle duplicate events
			return gorm.G[domain.ServerEvent](tx).Create(ctx, event)
		}

		if latestEvent.Status == event.Status {
			// ignore duplicate event
			return nil
		}

		return gorm.G[domain.ServerEvent](tx).Create(ctx, event)
	})
}
