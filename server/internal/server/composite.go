package server

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/authclient"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	importerhandler "github.com/minhnbnt/uptime-monitor/internal/features/importer/handler"
	notificationhandler "github.com/minhnbnt/uptime-monitor/internal/features/notification/handler"
	ontimehandler "github.com/minhnbnt/uptime-monitor/internal/features/ontime/handler"
	serverhandler "github.com/minhnbnt/uptime-monitor/internal/features/server/handler"
)

type CompositeHandler struct {
	*serverhandler.ServerHandler
	*serverhandler.EndpointHandler
	*importerhandler.ImportHandler
	*ontimehandler.OntimeHandler
	*authclient.AuthHandler
	*notificationhandler.NotificationHandler
	logger *slog.Logger
}

func RegisterCompositeHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*CompositeHandler, error) {
		return &CompositeHandler{
			ServerHandler:       do.MustInvoke[*serverhandler.ServerHandler](i),
			EndpointHandler:     do.MustInvoke[*serverhandler.EndpointHandler](i),
			ImportHandler:       do.MustInvoke[*importerhandler.ImportHandler](i),
			OntimeHandler:       do.MustInvoke[*ontimehandler.OntimeHandler](i),
			AuthHandler:         do.MustInvoke[*authclient.AuthHandler](i),
			NotificationHandler: do.MustInvoke[*notificationhandler.NotificationHandler](i),
			logger:              do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (h *CompositeHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {
	h.logger.Error("unhandled error", slog.Any("error", err))
	return apperrors.ToAPIError(err)
}
