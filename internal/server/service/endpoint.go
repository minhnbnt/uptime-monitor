package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/server/domain"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	repo "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/repository"
)

type EndpointService struct {
	endpointRepo *repo.EndpointRepository
}

func RegisterEndpointService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointService, error) {
		return &EndpointService{
			endpointRepo: do.MustInvoke[*repo.EndpointRepository](i),
		}, nil
	})
}

func toDomainEndpoint(serverID uint, req dto.SetCheckMethodRequest) domain.Endpoint {

	endpoint := domain.Endpoint{
		ServerID: serverID,
		Status:   domain.StatusActive,
		Interval: 30 * time.Second,
		Timeout:  10 * time.Second,
		Method:   "GET",
	}

	if req.URL != "" {
		endpoint.URL = req.URL
	}
	if req.Interval > 0 {
		endpoint.Interval = req.Interval
	}
	if req.Timeout > 0 {
		endpoint.Timeout = req.Timeout
	}
	if req.Method != "" {
		endpoint.Method = string(req.Method)
	}
	if req.ExpectedCode > 0 {
		endpoint.ExpectedCode = req.ExpectedCode
	}

	return endpoint
}

func (es *EndpointService) SetCheckMethod(ctx context.Context, serverID uint, req dto.SetCheckMethodRequest) error {

	endpoint := toDomainEndpoint(serverID, req)

	if err := es.endpointRepo.UpsertEndpoint(ctx, endpoint); err != nil {
		return fmt.Errorf("failed to upsert endpoint: %w", err)
	}

	// TODO: register scheduler

	return nil
}
