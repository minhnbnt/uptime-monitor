package server

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/handler"
	notificationhandler "github.com/minhnbnt/uptime-monitor/internal/features/notification/handler"
	featserverhandler "github.com/minhnbnt/uptime-monitor/internal/features/server/handler"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

type CompositeHandler struct {
	*featserverhandler.ServerHandler
	*featserverhandler.EndpointHandler
	*featserverhandler.ImportHandler
	*handler.AuthHandler
	*notificationhandler.NotificationHandler
	logger logger.Logger
}

func RegisterCompositeHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*CompositeHandler, error) {
		return &CompositeHandler{
			ServerHandler:       do.MustInvoke[*featserverhandler.ServerHandler](i),
			EndpointHandler:     do.MustInvoke[*featserverhandler.EndpointHandler](i),
			AuthHandler:         do.MustInvoke[*handler.AuthHandler](i),
			ImportHandler:       do.MustInvoke[*featserverhandler.ImportHandler](i),
			NotificationHandler: do.MustInvoke[*notificationhandler.NotificationHandler](i),
			logger:              do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (h *CompositeHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {
	h.logger.Error("unhandled error", logger.Error(err))
	return apperrors.ToAPIError(err)
}
