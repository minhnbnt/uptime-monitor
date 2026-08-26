package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/common/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/generated/api"
	pingservice "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/service"
)

type PushEventHandler struct {
	pushService *pingservice.PushEventService
	logger      *slog.Logger
}

func RegisterPushEventHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PushEventHandler, error) {
		return &PushEventHandler{
			pushService: do.MustInvoke[*pingservice.PushEventService](i),
			logger:      do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (h *PushEventHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {
	h.logger.Error("unhandled error", slog.Any("error", err))
	return &api.ErrorResponseStatusCode{
		StatusCode: http.StatusInternalServerError,
		Response: api.ErrorResponse{
			Error: api.ErrorResponseError{
				Code:    "INTERNAL_ERROR",
				Message: "internal error",
			},
		},
	}
}

func (h *PushEventHandler) PushEvents(ctx context.Context, req []api.PushEventItem) (api.PushEventsRes, error) {

	sessionID := authclient.GetSessionID(ctx)
	if sessionID == "" {
		h.logger.Warn("push rejected without session id", slog.Uint64("user", uint64(authclient.GetUserID(ctx))))
		return &api.PushEventsForbidden{}, nil
	}

	items := make([]pingservice.PushEventItem, len(req))
	for i, item := range req {
		items[i] = pingservice.PushEventItem{
			ID:     uint64(item.ID),
			Status: item.Status,
		}
	}

	result, err := h.pushService.Handle(ctx, authclient.GetUserID(ctx), sessionID, items)

	var rateLimited *pingservice.RateLimitedError
	switch {
	case errors.As(err, &rateLimited):
		return &api.RateLimitResponse{NextTime: rateLimited.NextTime.UnixMilli()}, nil
	case err != nil:
		return nil, err
	}

	eventErrors := lo.Map(result.Errors, func(e pingservice.PushEventError, _ int) api.PushEventError {
		return api.PushEventError{ID: int64(e.ID), Error: e.Error}
	})
	accepted := lo.Map(result.Accepted, func(id uint64, _ int) int64 {
		return int64(id)
	})

	body := api.PushEventsResponse{
		NextTime: result.NextTime.UnixMilli(),
		StaleAt:  result.StaleAt.UnixMilli(),
		Accepted: accepted,
		Errors:   eventErrors,
	}

	switch {
	case len(accepted) > 0 && len(eventErrors) == 0:
		return (*api.PushEventsOK)(&body), nil
	case len(accepted) > 0:
		return (*api.PushEventsMultiStatus)(&body), nil
	default:
		return &api.PushErrorsResponse{Errors: eventErrors}, nil
	}
}
