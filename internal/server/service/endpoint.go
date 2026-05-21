package service

import (
	"context"
	"fmt"

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

	if err := es.endpointRepo.UpsertEndpoint(ctx, endpoint); err != nil {
		return fmt.Errorf("failed to upsert endpoint: %w", err)
	}

	return nil
}
