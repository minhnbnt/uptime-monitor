package service

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	infra "github.com/minhnbnt/uptime-monitor/internal/monitor/infrastructure"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

type EndpointService struct {
	endpointRepository EndpointRepository
	pingWorker         *infra.PingWorker
	logger             logger.Logger
}

func RegisterEndpointService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointService, error) {
		return &EndpointService{
			endpointRepository: do.MustInvoke[*serverrepo.EndpointRepository](i),
			pingWorker:         do.MustInvoke[*infra.PingWorker](i),
			logger:             do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func toDomainEndpoint(serverID uint, req dto.SetCheckMethodRequest) domain.Endpoint {
	return domain.Endpoint{
		ServerID:     serverID,
		Status:       domain.StatusActive,
		URL:          req.URL,
		Interval:     req.Interval,
		Timeout:      req.Timeout,
		Method:       req.HTTPMethod,
		ExpectedCode: req.ExpectedCode,
	}
}

func (es *EndpointService) SetCheckMethod(ctx context.Context, serverID uint, req dto.SetCheckMethodRequest) error {

	endpoint := toDomainEndpoint(serverID, req)

	if err := es.endpointRepository.UpsertEndpoint(ctx, endpoint); err != nil {
		es.logger.Error("failed to upsert endpoint", logger.Error(err))
		return apperrors.ErrInternal
	}

	return nil
}

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
