package service

import (
	"context"
	"errors"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	infra "github.com/minhnbnt/uptime-monitor/internal/features/ping/infrastructure"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

type Pinger interface {
	Ping(ctx context.Context, method, url string) (statusCode int, err error)
}

type EndpointService struct {
	serverRepository   ServerRepository
	endpointRepository EndpointRepository
	pingWorker         Pinger
	logger             logger.Logger
}

func RegisterEndpointService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointService, error) {
		return &EndpointService{
			serverRepository:   do.MustInvoke[*serverrepo.ServerRepository](i),
			endpointRepository: do.MustInvoke[*serverrepo.EndpointRepository](i),
			pingWorker:         do.MustInvoke[*infra.PingWorker](i),
			logger:             do.MustInvoke[logger.Logger](i),
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
		es.logger.Error("failed to get server for set check method", logger.Error(err))
		return apperrors.ErrInternal
	}

	if server.CreatedByID != userID {
		return apperrors.ErrForbidden
	}

	endpoint := toDomainEndpoint(serverID, req)

	if err := es.endpointRepository.UpsertEndpoint(ctx, endpoint); err != nil {
		es.logger.Error("failed to upsert endpoint", logger.Error(err))
		return apperrors.ErrInternal
	}

	return nil
}

var _ Pinger = (*infra.PingWorker)(nil)

func (es *EndpointService) TestEndpoint(ctx context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error) {

	pingCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	statusCode, err := es.pingWorker.Ping(pingCtx, req.Method, req.URL)
	if err != nil {
		errMsg := err.Error()
		return &dto.TestEndpointResponse{
			Success:    false,
			StatusCode: 0,
			Error:      &errMsg,
		}, nil
	}

	return &dto.TestEndpointResponse{
		Success:    statusCode == req.ExpectedCode,
		StatusCode: statusCode,
	}, nil
}
