package server

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/handler"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	serverhandler "github.com/minhnbnt/uptime-monitor/internal/server/handler"
)

type CompositeHandler struct {
	*serverhandler.ServerHandler
	*serverhandler.EndpointHandler
	*handler.AuthHandler
	*serverhandler.ImportHandler
	*serverhandler.NotificationHandler
	logger logger.Logger
}

func RegisterCompositeHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*CompositeHandler, error) {
		return &CompositeHandler{
			ServerHandler:       do.MustInvoke[*serverhandler.ServerHandler](i),
			EndpointHandler:     do.MustInvoke[*serverhandler.EndpointHandler](i),
			AuthHandler:         do.MustInvoke[*handler.AuthHandler](i),
			ImportHandler:       do.MustInvoke[*serverhandler.ImportHandler](i),
			NotificationHandler: do.MustInvoke[*serverhandler.NotificationHandler](i),
			logger:              do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (h *CompositeHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {
	h.logger.Error("unhandled error", logger.Error(err))
	return apperrors.ToAPIError(err)
}
