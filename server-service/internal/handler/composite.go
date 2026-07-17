package handler

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
)

type CompositeHandler struct {
	*ServerHandler
	*EndpointHandler
	logger *slog.Logger
}

func RegisterCompositeHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*CompositeHandler, error) {
		return &CompositeHandler{
			ServerHandler:   do.MustInvoke[*ServerHandler](i),
			EndpointHandler: do.MustInvoke[*EndpointHandler](i),
			logger:          do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (h *CompositeHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {
	h.logger.Error("unhandled error", slog.Any("error", err))
	return apperrors.ToAPIError(err)
}
