package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/common/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/service"
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

	dtoReq := dto.SetCheckMethodRequest{Method: dto.CheckMethodType(req.Method)}

	if req.Method == api.CheckMethodTypePull {
		ep, ok := req.Endpoint.Get()
		if !ok {
			return nil, &api.ErrorResponseStatusCode{
				StatusCode: http.StatusBadRequest,
				Response: api.ErrorResponse{
					Error: api.ErrorResponseError{
						Code:    "INVALID_REQUEST",
						Message: "endpoint is required for the pull check method",
					},
				},
			}
		}
		dtoReq.HTTPMethod = string(ep.Method)
		dtoReq.Interval = time.Duration(ep.Interval) * time.Second
		dtoReq.Timeout = time.Duration(ep.Timeout) * time.Second
		dtoReq.URL = ep.URL.String()
		dtoReq.ExpectedCode = ep.ExpectedCode

		if v, ok := ep.BodyCheckExpr.Get(); ok {
			dtoReq.BodyCheckExpr = &v
		}
	}

	userID := authclient.GetUserID(ctx)
	err := h.endpointService.SetCheckMethod(ctx, uint(params.ID), userID, dtoReq)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	server, err := h.serverService.GetServer(ctx, uint(params.ID))
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.ServerResponse{Data: ToAPIServer(server)}, nil
}

var _ EndpointService = (*service.EndpointService)(nil)

func (h *EndpointHandler) TestEndpoint(
	ctx context.Context,
	req *api.TestEndpointRequest,
) (*api.TestEndpointResponse, error) {

	timeout := req.Timeout.Or(10)
	expectedCode := req.ExpectedCode.Or(200)

	dtoReq := dto.TestEndpointRequest{
		URL:          req.URL.String(),
		Method:       string(req.Method),
		Timeout:      time.Duration(timeout) * time.Second,
		ExpectedCode: expectedCode,
	}

	if v, ok := req.BodyCheckExpr.Get(); ok {
		bodyExpr := v
		dtoReq.BodyCheckExpr = &bodyExpr
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
