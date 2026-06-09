package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
)

type EndpointHandler struct {
	endpointService EndpointService
	serverService   ServerService
}

func RegisterEndpointHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointHandler, error) {
		return &EndpointHandler{
			endpointService: do.MustInvoke[*service.EndpointService](i),
			serverService:   do.MustInvoke[*service.ServerService](i),
		}, nil
	})
}

func (h *EndpointHandler) SetCheckMethod(
	ctx context.Context, req *api.SetCheckMethodRequest, params api.SetCheckMethodParams,
) (*api.ServerResponse, error) {

	dtoReq := dto.SetCheckMethodRequest{
		Method:       dto.CheckMethodType(req.Method),
		HTTPMethod:   req.Endpoint.Method,
		Interval:     time.Duration(req.Endpoint.Interval) * time.Second,
		Timeout:      time.Duration(req.Endpoint.Timeout) * time.Second,
		URL:          req.Endpoint.URL.String(),
		ExpectedCode: req.Endpoint.ExpectedCode,
	}

	if err := h.endpointService.SetCheckMethod(ctx, uint(params.ID), dtoReq); err != nil {
		return nil, &api.ErrorResponseStatusCode{
			StatusCode: http.StatusInternalServerError,
			Response:   errResponse("INTERNAL_ERROR", err.Error()),
		}
	}

	server, err := h.serverService.GetServer(ctx, uint(params.ID))
	if err != nil {
		return nil, &api.ErrorResponseStatusCode{
			StatusCode: http.StatusNotFound,
			Response:   errResponse("NOT_FOUND", "Server not found"),
		}
	}

	return &api.ServerResponse{Data: toAPIServer(server)}, nil
}
