package server

import (
	"context"
	"net/http"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/server/handler"
)

type CompositeHandler struct {
	*handler.ServerHandler
	*handler.EndpointHandler
	*handler.AuthHandler
}

func RegisterCompositeHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*CompositeHandler, error) {
		return &CompositeHandler{
			ServerHandler:   do.MustInvoke[*handler.ServerHandler](i),
			EndpointHandler: do.MustInvoke[*handler.EndpointHandler](i),
			AuthHandler:     do.MustInvoke[*handler.AuthHandler](i),
		}, nil
	})
}

func (h *CompositeHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {
	return &api.ErrorResponseStatusCode{
		StatusCode: http.StatusInternalServerError,
		Response: api.ErrorResponse{
			Error: api.ErrorResponseError{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		},
	}
}
