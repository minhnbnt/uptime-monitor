package handler

import (
	"context"
	"io"

	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

type ImportService interface {
	ImportServers(ctx context.Context, userID uint, file io.Reader) (*dto.ImportResult, error)
	GenerateTemplate(w io.Writer) error
}

type NotificationService interface {
	GetNotificationConfig(ctx context.Context, userID uint) (*dto.NotificationConfigResponse, error)
	UpdateNotificationConfig(ctx context.Context, userID uint, req *dto.NotificationConfigRequest) error
	SendReport(ctx context.Context, userID uint) error
}
