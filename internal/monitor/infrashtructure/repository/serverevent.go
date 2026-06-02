package repository

import (
	"context"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/monitor/domain"
)

type ServerEventRepository struct {
	db *gorm.DB
}

func newServerEventRepository(i do.Injector) (*ServerEventRepository, error) {
	dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
	return &ServerEventRepository{db: dbWrapper.GetDB()}, nil
}

func RegisterServerEventRepository(i do.Injector) {
	do.Provide(i, newServerEventRepository)
}

func (r *ServerEventRepository) Save(ctx context.Context, event *domain.ServerEvent) error {
	return gorm.G[domain.ServerEvent](r.db).Create(ctx, event)
}
