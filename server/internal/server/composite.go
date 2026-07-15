package server

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	importerhandler "github.com/minhnbnt/uptime-monitor/internal/features/importer/handler"
	ontimehandler "github.com/minhnbnt/uptime-monitor/internal/features/ontime/handler"
	serverhandler "github.com/minhnbnt/uptime-monitor/internal/features/server/handler"
)

type CompositeHandler struct {
	*serverhandler.ServerHandler
	*serverhandler.EndpointHandler
	*importerhandler.ImportHandler
	*ontimehandler.OntimeHandler
	logger *slog.Logger
}

func RegisterCompositeHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*CompositeHandler, error) {
		return &CompositeHandler{
			ServerHandler:       do.MustInvoke[*serverhandler.ServerHandler](i),
			EndpointHandler:     do.MustInvoke[*serverhandler.EndpointHandler](i),
			ImportHandler:       do.MustInvoke[*importerhandler.ImportHandler](i),
			OntimeHandler:       do.MustInvoke[*ontimehandler.OntimeHandler](i),
			logger:              do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (h *CompositeHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {
	h.logger.Error("unhandled error", slog.Any("error", err))
	return apperrors.ToAPIError(err)
}
