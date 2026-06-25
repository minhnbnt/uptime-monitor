package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/features/notification/dto"
)

type mockNotificationService struct {
	getNotificationConfigFn    func(ctx context.Context, userID uint) (*dto.NotificationConfigResponse, error)
	updateNotificationConfigFn func(ctx context.Context, userID uint, req *dto.NotificationConfigRequest) error
	sendReportFn               func(ctx context.Context, userID uint) error
}

func (m *mockNotificationService) GetNotificationConfig(ctx context.Context, userID uint) (*dto.NotificationConfigResponse, error) {
	return m.getNotificationConfigFn(ctx, userID)
}

func (m *mockNotificationService) UpdateNotificationConfig(ctx context.Context, userID uint, req *dto.NotificationConfigRequest) error {
	return m.updateNotificationConfigFn(ctx, userID, req)
}

func (m *mockNotificationService) SendReport(ctx context.Context, userID uint) error {
	return m.sendReportFn(ctx, userID)
}
