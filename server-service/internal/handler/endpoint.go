package handler

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/service"
)

type EndpointHandler struct {
	endpointService EndpointService
}

func RegisterEndpointHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointHandler, error) {
		return &EndpointHandler{
			endpointService: do.MustInvoke[*service.EndpointService](i),
		}, nil
	})
}

func (h *EndpointHandler) TestEndpoint(
	ctx context.Context,
	req *api.TestEndpointRequest,
) (*api.TestEndpointResponse, error) {

	dto := ToTestEndpointRequest(req)
	result, err := h.endpointService.TestEndpoint(ctx, dto)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	resp := &api.TestEndpointResponse{
		Running: result.Running,
	}

	if result.Error != nil {
		resp.Error = api.NewOptString(*result.Error)
	}

	return resp, nil
}

var _ EndpointService = (*service.EndpointService)(nil)
