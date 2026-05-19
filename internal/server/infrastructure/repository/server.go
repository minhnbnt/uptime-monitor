package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/server/domain"
)

type ServerRepository struct {
	db *gorm.DB
}

func RegisterServerRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerRepository, error) {
		return &ServerRepository{db: do.MustInvoke[*gorm.DB](i)}, nil
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

func (sr *ServerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Server, error) {

	server, err := gorm.G[domain.Server](sr.db).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return &server, nil
}

func (sr *ServerRepository) Update(ctx context.Context, s *domain.Server) error {
	_, err := gorm.G[domain.Server](sr.db).Where("id = ?", s.ID).Updates(ctx, *s)
	return err
}

func (sr *ServerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := gorm.G[domain.Server](sr.db).Where("id = ?", id).Delete(ctx)
	return err
}
