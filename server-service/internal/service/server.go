package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/repository"
)

type ServerWriter interface {
	Create(ctx context.Context, s *domain.Server, config *domain.ServerHttpConfig) error
	Update(ctx context.Context, s *domain.Server, config *domain.ServerHttpConfig) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*domain.Server, error)
}

type ServerService struct {
	*ServerReader
	serverWriter ServerWriter
	serverRepo   *repository.ServerRepository
}

func RegisterServerService(i do.Injector) {

	do.Provide(i, func(i do.Injector) (*ServerService, error) {
		return &ServerService{
			ServerReader: do.MustInvoke[*ServerReader](i),
			serverWriter: do.MustInvoke[*repository.ServerRepository](i),
			serverRepo:   do.MustInvoke[*repository.ServerRepository](i),
		}, nil
	})
}

func (ss *ServerService) CountByStatus(
	ctx context.Context, userID uint,
) (total, online, offline int64, err error) {
	return ss.serverRepo.CountByStatus(ctx, userID)
}

func (ss *ServerService) CreateServer(
	ctx context.Context,
	req dto.CreateServerRequest,
	createdByID uint,
) (*dto.Server, error) {

	server := domain.Server{
		Name:          req.Name,
		Namespace:     req.Namespace,
		Kind:          req.Kind,
		ObjectID:      req.ObjectID,
		ContainerName: req.ContainerName,
		Interval:      req.Interval,
		Timeout:       req.Timeout,
		CreatedByID:   createdByID,
	}

	var config *domain.ServerHttpConfig
	if req.HttpConfig != nil {
		config = &domain.ServerHttpConfig{
			ServerID:      server.ID,
			Port:          req.HttpConfig.Port,
			EndpointPath:  req.HttpConfig.EndpointPath,
			ExpectedCode:  req.HttpConfig.ExpectedCode,
			BodyCheckExpr: req.HttpConfig.BodyCheckExpr,
			Method:        defaultHttpMethod(req.HttpConfig.Method),
		}
	}

	if err := ss.serverWriter.Create(ctx, &server, config); err != nil {
		ss.logger.Error("failed to create server", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	result := dto.ServerFromDomain(server)
	return &result, nil
}

func (ss *ServerService) UpdateServer(ctx context.Context, id uint, userID uint, req dto.UpdateServerRequest) (*dto.Server, error) {

	server, err := ss.serverWriter.GetByID(ctx, id)
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.ErrNotFound
	}
	if err != nil {
		ss.logger.Error("failed to get server for update", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	if server.CreatedByID != userID {
		return nil, apperrors.ErrForbidden
	}

	applyUpdateServer(server, req)

	var config *domain.ServerHttpConfig
	if req.HttpConfig != nil {
		config = &domain.ServerHttpConfig{
			ServerID:      server.ID,
			Port:          req.HttpConfig.Port,
			EndpointPath:  req.HttpConfig.EndpointPath,
			ExpectedCode:  req.HttpConfig.ExpectedCode,
			BodyCheckExpr: req.HttpConfig.BodyCheckExpr,
			Method:        defaultHttpMethod(req.HttpConfig.Method),
		}
	}

	err = ss.serverWriter.Update(ctx, server, config)
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.ErrNotFound
	}

	if err != nil {
		ss.logger.Error("failed to update server", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	result := dto.ServerFromDomain(*server)
	return &result, nil
}

func (ss *ServerService) DeleteServer(ctx context.Context, id uint, userID uint) error {

	server, err := ss.serverWriter.GetByID(ctx, id)
	if errors.Is(err, apperrors.ErrNotFound) {
		return apperrors.ErrNotFound
	}
	if err != nil {
		ss.logger.Error("failed to get server", slog.Any("error", err))
		return apperrors.ErrInternal
	}

	if server.CreatedByID != userID {
		return apperrors.ErrForbidden
	}

	err = ss.serverWriter.Delete(ctx, id)
	if errors.Is(err, apperrors.ErrNotFound) {
		return apperrors.ErrNotFound
	}
	if err != nil {
		ss.logger.Error("failed to delete server", slog.Any("error", err))
		return apperrors.ErrInternal
	}

	return nil
}

func applyUpdateServer(s *domain.Server, req dto.UpdateServerRequest) {

	if req.Name != nil {
		s.Name = *req.Name
	}
	if req.Namespace != nil {
		s.Namespace = *req.Namespace
	}
	if req.Kind != nil {
		s.Kind = *req.Kind
	}
	if req.ObjectID != nil {
		s.ObjectID = *req.ObjectID
	}
	if req.ContainerName != nil {
		s.ContainerName = *req.ContainerName
	}
	if req.Interval != nil {
		s.Interval = *req.Interval
	}
	if req.Timeout != nil {
		s.Timeout = *req.Timeout
	}
}

func defaultHttpMethod(method string) string {
	if method == "" {
		return "GET"
	}
	return method
}
