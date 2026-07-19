package handler

import (
	"context"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/common/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/service"
)

type NotificationService interface {
	GetNotificationConfig(ctx context.Context, userID uint) (*dto.NotificationConfigResponse, error)
	UpdateNotificationConfig(ctx context.Context, userID uint, req *dto.NotificationConfigRequest) error
	SendReport(ctx context.Context, userID uint) error
}

type NotificationHandler struct {
	notificationService NotificationService
}

func RegisterNotificationHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*NotificationHandler, error) {
		return &NotificationHandler{
			notificationService: do.MustInvoke[*service.NotificationService](i),
		}, nil
	})
}

func (h *NotificationHandler) GetNotificationConfig(ctx context.Context) (*api.NotificationConfig, error) {

	userID := authclient.GetUserID(ctx)
	cfg, err := h.notificationService.GetNotificationConfig(ctx, userID)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	resp := &api.NotificationConfig{
		DigestTime: api.NewOptString(cfg.DigestTime),
	}

	if cfg.FromDate != "" {
		fromDate, err := time.Parse("2006-01-02", cfg.FromDate)
		if err == nil {
			resp.FromDate = api.NewOptDate(fromDate)
		}
	}

	if cfg.ToDate != "" {
		toDate, err := time.Parse("2006-01-02", cfg.ToDate)
		if err == nil {
			resp.ToDate = api.NewOptDate(toDate)
		}
	}

	return resp, nil
}

func (h *NotificationHandler) UpdateNotificationConfig(ctx context.Context, req *api.NotificationConfig) error {

	userID := authclient.GetUserID(ctx)

	dtoReq := &dto.NotificationConfigRequest{}
	if req.FromDate.Set {
		dtoReq.FromDate = req.FromDate.Value.Format("2006-01-02")
	}
	if req.ToDate.Set {
		dtoReq.ToDate = req.ToDate.Value.Format("2006-01-02")
	}
	if req.DigestTime.Set {
		dtoReq.DigestTime = req.DigestTime.Value
	}

	if err := h.notificationService.UpdateNotificationConfig(ctx, userID, dtoReq); err != nil {
		return apperrors.ToAPIError(err)
	}

	return nil
}

var _ NotificationService = (*service.NotificationService)(nil)

func (h *NotificationHandler) SendReport(ctx context.Context) error {

	userID := authclient.GetUserID(ctx)
	if err := h.notificationService.SendReport(ctx, userID); err != nil {
		return apperrors.ToAPIError(err)
	}

	return nil
}

func (h *NotificationHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {
	return apperrors.ToAPIError(err)
}
