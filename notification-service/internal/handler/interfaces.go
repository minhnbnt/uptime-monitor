package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/dto"
)

type NotificationService interface {
	GetNotificationConfig(ctx context.Context, userID uint) (*dto.NotificationConfigResponse, error)
	UpdateNotificationConfig(ctx context.Context, userID uint, req *dto.NotificationConfigRequest) error
	SendReport(ctx context.Context, userID uint) error
}
