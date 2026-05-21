package repository

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/server/domain"
)

type ServerRepository struct {
	db *gorm.DB
}

func RegisterServerRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return &ServerRepository{db: dbWrapper.GetDB()}, nil
	})
}

func (sr *ServerRepository) List(ctx context.Context, limit, offset int) ([]domain.Server, error) {

	servers, err := gorm.G[domain.Server](sr.db).
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

func (sr *ServerRepository) GetByID(ctx context.Context, id uint) (*domain.Server, error) {

	server, err := gorm.G[domain.Server](sr.db).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return &server, nil
}

func (sr *ServerRepository) Update(ctx context.Context, s *domain.Server) error {

	rowAffected, err := gorm.G[domain.Server](sr.db).Where("id = ?", s.ID).Updates(ctx, *s)
	if err != nil {
		return err
	}

	if rowAffected == 0 {
		return fmt.Errorf("server with id %d does not found", s.ID)
	}

	return nil
}

func (sr *ServerRepository) Delete(ctx context.Context, id uint) error {

	rowAffected, err := gorm.G[domain.Server](sr.db).Where("id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}

	if rowAffected == 0 {
		return fmt.Errorf("server with id %d does not found", id)
	}

	return nil
}
