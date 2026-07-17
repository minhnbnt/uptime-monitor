package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/repository"
)

type EndpointService struct {
	serverRepository   ServerRepository
	endpointRepository EndpointRepository
	logger             *slog.Logger
}

func RegisterEndpointService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointService, error) {
		return &EndpointService{
			serverRepository:   do.MustInvoke[*repository.ServerRepository](i),
			endpointRepository: do.MustInvoke[*repository.EndpointRepository](i),
			logger:             do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func toDomainEndpoint(serverID uint, req dto.SetCheckMethodRequest) domain.Endpoint {
	return domain.Endpoint{
		ServerID:      serverID,
		URL:           req.URL,
		MonitorStatus: domain.StatusOff,
		Interval:      req.Interval,
		Timeout:       req.Timeout,
		Method:        req.HTTPMethod,
		ExpectedCode:  req.ExpectedCode,
	}
}

func (es *EndpointService) SetCheckMethod(ctx context.Context, serverID uint, userID uint, req dto.SetCheckMethodRequest) error {

	server, err := es.serverRepository.GetByID(ctx, serverID)
	if errors.Is(err, apperrors.ErrNotFound) {
		return apperrors.ErrNotFound
	}
	if err != nil {
		es.logger.Error("failed to get server for set check method", slog.Any("error", err))
		return apperrors.ErrInternal
	}

	if server.CreatedByID != userID {
		return apperrors.ErrForbidden
	}

	endpoint := toDomainEndpoint(serverID, req)

	if err := es.endpointRepository.UpsertEndpoint(ctx, endpoint); err != nil {
		es.logger.Error("failed to upsert endpoint", slog.Any("error", err))
		return apperrors.ErrInternal
	}

	return nil
}

func (es *EndpointService) TestEndpoint(ctx context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error) {

	pingCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	statusCode, err := infrastructure.PingURL(pingCtx, req.Method, req.URL)
	if err != nil {
		return &dto.TestEndpointResponse{
			Success:    false,
			StatusCode: 0,
			Error:      new(err.Error()),
		}, nil
	}

	return &dto.TestEndpointResponse{
		Success:    statusCode == req.ExpectedCode,
		StatusCode: statusCode,
	}, nil
}
