package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/dto"
)

type stubNotificationService struct {
	config *dto.NotificationConfigResponse
	err    error
	lastUpsert *dto.NotificationConfigRequest
}

func (s *stubNotificationService) GetNotificationConfig(_ context.Context, _ uint) (*dto.NotificationConfigResponse, error) {
	return s.config, s.err
}

func (s *stubNotificationService) UpdateNotificationConfig(_ context.Context, _ uint, req *dto.NotificationConfigRequest) error {
	s.lastUpsert = req
	return s.err
}

func (s *stubNotificationService) SendReport(_ context.Context, _ uint) error {
	return s.err
}

func TestGetNotificationConfig_MapsTimezone(t *testing.T) {
	stub := &stubNotificationService{
		config: &dto.NotificationConfigResponse{DigestTime: "09:00", Timezone: "Asia/Saigon"},
	}
	h := &NotificationHandler{notificationService: stub}

	resp, err := h.GetNotificationConfig(t.Context())
	require.NoError(t, err)
	require.Equal(t, "09:00", resp.DigestTime.Value)
	require.True(t, resp.Timezone.IsSet())
	require.Equal(t, "Asia/Saigon", resp.Timezone.Value)
}

func TestUpdateNotificationConfig_MapsTimezone(t *testing.T) {
	stub := &stubNotificationService{}
	h := &NotificationHandler{notificationService: stub}

	err := h.UpdateNotificationConfig(t.Context(), &api.NotificationConfig{
		DigestTime: api.NewOptString("09:00"),
		Timezone:   api.NewOptString("Asia/Saigon"),
	})
	require.NoError(t, err)
	require.NotNil(t, stub.lastUpsert)
	require.Equal(t, "Asia/Saigon", stub.lastUpsert.Timezone)
	require.Equal(t, "09:00", stub.lastUpsert.DigestTime)
}

func TestUpdateNotificationConfig_OmitsTimezoneWhenUnset(t *testing.T) {
	stub := &stubNotificationService{}
	h := &NotificationHandler{notificationService: stub}

	err := h.UpdateNotificationConfig(t.Context(), &api.NotificationConfig{
		DigestTime: api.NewOptString("09:00"),
	})
	require.NoError(t, err)
	require.NotNil(t, stub.lastUpsert)
	require.Equal(t, "", stub.lastUpsert.Timezone)
}
