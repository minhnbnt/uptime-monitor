package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/internal/server/domain"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	repo "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/repository"
)

type ServerService struct {
	repo *repo.ServerRepository
}

func RegisterServerService(i do.Injector) {

	do.Provide(i, func(i do.Injector) (*ServerService, error) {
		return &ServerService{
			repo: do.MustInvoke[*repo.ServerRepository](i),
		}, nil
	})
}

func (ss *ServerService) ListServers(ctx context.Context, page, perPage int) ([]dto.Server, error) {

	limit, offset := perPage, (page-1)*perPage

	result, err := ss.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get servers: %w", err)
	}

	return lo.Map(result, func(item domain.Server, index int) dto.Server {
		return dto.Server{
			ID:        item.ID,
			Name:      item.Name,
			URL:       item.URL,
			Status:    item.Status,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		}
	}), nil
}

func (ss *ServerService) CreateServer(ctx context.Context, req dto.CreateServerRequest) (*dto.Server, error) {

	server := domain.Server{
		Name:   req.Name,
		URL:    req.URL,
		Status: domain.StatusActive,
	}

	if err := ss.repo.Create(ctx, &server); err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	return &dto.Server{
		ID:        server.ID,
		Name:      server.Name,
		URL:       server.URL,
		Status:    server.Status,
		CreatedAt: server.CreatedAt,
		UpdatedAt: server.UpdatedAt,
	}, nil
}

func (ss *ServerService) GetServer(ctx context.Context, id uuid.UUID) (*dto.Server, error) {

	server, err := ss.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return &dto.Server{
		ID:        server.ID,
		Name:      server.Name,
		URL:       server.URL,
		Status:    server.Status,
		CreatedAt: server.CreatedAt,
		UpdatedAt: server.UpdatedAt,
	}, nil
}

func (ss *ServerService) UpdateServer(ctx context.Context, id uuid.UUID, req dto.UpdateServerRequest) (*dto.Server, error) {

	server, err := ss.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	if req.Name != nil {
		server.Name = *req.Name
	}
	if req.URL != nil {
		server.URL = *req.URL
	}
	if req.Status != nil {
		server.Status = *req.Status
	}

	if err := ss.repo.Update(ctx, server); err != nil {
		return nil, fmt.Errorf("failed to update server: %w", err)
	}

	return &dto.Server{
		ID:        server.ID,
		Name:      server.Name,
		URL:       server.URL,
		Status:    server.Status,
		CreatedAt: server.CreatedAt,
		UpdatedAt: server.UpdatedAt,
	}, nil
}

func (ss *ServerService) DeleteServer(ctx context.Context, id uuid.UUID) error {

	if err := ss.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	return nil
}
