package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
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

	if req.Method == api.CheckMethodTypePush {
		return nil, &api.ErrorResponseStatusCode{
			StatusCode: http.StatusNotImplemented,
			Response: api.ErrorResponse{
				Error: api.ErrorResponseError{
					Code:    "NOT_IMPLEMENTED",
					Message: "Push check method is not yet implemented",
				},
			},
		}
	}

	dtoReq := dto.SetCheckMethodRequest{
		Method:       dto.CheckMethodType(req.Method),
		HTTPMethod:   string(req.Endpoint.Method),
		Interval:     time.Duration(req.Endpoint.Interval) * time.Second,
		Timeout:      time.Duration(req.Endpoint.Timeout) * time.Second,
		URL:          req.Endpoint.URL.String(),
		ExpectedCode: req.Endpoint.ExpectedCode,
	}

	if err := h.endpointService.SetCheckMethod(ctx, uint(params.ID), dtoReq); err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	server, err := h.serverService.GetServer(ctx, uint(params.ID))
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.ServerResponse{Data: toAPIServer(server)}, nil
}

func (h *EndpointHandler) TestEndpoint(ctx context.Context, req *api.TestEndpointRequest) (*api.TestEndpointResponse, error) {

	timeout := req.Timeout.Or(10)
	expectedCode := req.ExpectedCode.Or(200)

	dtoReq := dto.TestEndpointRequest{
		URL:          req.URL.String(),
		Method:       string(req.Method),
		Timeout:      time.Duration(timeout) * time.Second,
		ExpectedCode: expectedCode,
	}

	result, err := h.endpointService.TestEndpoint(ctx, dtoReq)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	resp := &api.TestEndpointResponse{
		Success:    result.Success,
		StatusCode: result.StatusCode,
	}

	if result.Error != nil {
		resp.Error = api.NewOptString(*result.Error)
	}

	return resp, nil
}
