package server

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/handler"
	importerhandler "github.com/minhnbnt/uptime-monitor/internal/features/importer/handler"
	notificationhandler "github.com/minhnbnt/uptime-monitor/internal/features/notification/handler"
	ontimehandler "github.com/minhnbnt/uptime-monitor/internal/features/ontime/handler"
	featserverhandler "github.com/minhnbnt/uptime-monitor/internal/features/server/handler"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

type CompositeHandler struct {
	*featserverhandler.ServerHandler
	*featserverhandler.EndpointHandler
	*importerhandler.ImportHandler
	*ontimehandler.OntimeHandler
	*handler.AuthHandler
	*notificationhandler.NotificationHandler
	logger logger.Logger
}

func RegisterCompositeHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*CompositeHandler, error) {
		return &CompositeHandler{
			ServerHandler:       do.MustInvoke[*featserverhandler.ServerHandler](i),
			EndpointHandler:     do.MustInvoke[*featserverhandler.EndpointHandler](i),
			ImportHandler:       do.MustInvoke[*importerhandler.ImportHandler](i),
			OntimeHandler:       do.MustInvoke[*ontimehandler.OntimeHandler](i),
			AuthHandler:         do.MustInvoke[*handler.AuthHandler](i),
			NotificationHandler: do.MustInvoke[*notificationhandler.NotificationHandler](i),
			logger:              do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (h *CompositeHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {
	h.logger.Error("unhandled error", logger.Error(err))
	return apperrors.ToAPIError(err)
}
