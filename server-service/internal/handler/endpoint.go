package handler

import (
	"context"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
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

	timeout := req.Timeout.Or(10)

	dtoReq := dto.TestEndpointRequest{
		Namespace: req.Namespace,
		ObjectID:  req.ObjectID,
		Kind:      string(req.Kind),
		Timeout:   time.Duration(timeout) * time.Second,
	}

	if v, ok := req.ContainerName.Get(); ok {
		dtoReq.ContainerName = v
	}

	result, err := h.endpointService.TestEndpoint(ctx, dtoReq)
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
