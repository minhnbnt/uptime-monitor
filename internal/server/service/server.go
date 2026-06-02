package service

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
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

func toDTOEndpoint(e *domain.Endpoint) *dto.Endpoint {
	if e == nil {
		return nil
	}
	return &dto.Endpoint{
		URL:          e.URL,
		Status:       domain.Status(e.Status),
		Interval:     e.Interval,
		Timeout:      e.Timeout,
		Method:       e.Method,
		ExpectedCode: e.ExpectedCode,
	}
}

func toDTOServer(s domain.Server) dto.Server {
	return dto.Server{
		ID:        s.ID,
		Name:      s.Name,
		Status:    s.Status,
		Endpoint:  toDTOEndpoint(s.Endpoint),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func (ss *ServerService) ListServers(ctx context.Context, page, perPage int) ([]dto.Server, error) {

	limit, offset := perPage, (page-1)*perPage

	result, err := ss.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get servers: %w", err)
	}

	return lo.Map(result, func(item domain.Server, index int) dto.Server {
		return toDTOServer(item)
	}), nil
}

func (ss *ServerService) CreateServer(ctx context.Context, req dto.CreateServerRequest) (*dto.Server, error) {

	server := domain.Server{
		Name:   req.Name,
		Status: domain.StatusActive,
	}

	if err := ss.repo.Create(ctx, &server); err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	result := toDTOServer(server)
	return &result, nil
}

func (ss *ServerService) GetServer(ctx context.Context, id uint) (*dto.Server, error) {

	server, err := ss.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	result := toDTOServer(*server)
	return &result, nil
}

func (ss *ServerService) UpdateServer(ctx context.Context, id uint, req dto.UpdateServerRequest) (*dto.Server, error) {

	server, err := ss.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	if req.Name != nil {
		server.Name = *req.Name
	}

	if req.Status != nil {
		server.Status = *req.Status
	}

	if err := ss.repo.Update(ctx, server); err != nil {
		return nil, fmt.Errorf("failed to update server: %w", err)
	}

	result := toDTOServer(*server)
	return &result, nil
}

func (ss *ServerService) DeleteServer(ctx context.Context, id uint) error {

	if err := ss.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	return nil
}
