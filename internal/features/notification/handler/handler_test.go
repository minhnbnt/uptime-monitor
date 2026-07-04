package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/notification/dto"
)

func TestNotificationHandler_GetNotificationConfig(t *testing.T) {
	t.Run("success with dates", func(t *testing.T) {
		h := &NotificationHandler{
			notificationService: &mockNotificationService{
				getNotificationConfigFn: func(_ context.Context, userID uint) (*dto.NotificationConfigResponse, error) {
					return &dto.NotificationConfigResponse{
						FromDate:   "2026-06-01",
						ToDate:     "2026-06-30",
						DigestTime: "09:00",
					}, nil
				},
			},
		}

		resp, err := h.GetNotificationConfig(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.DigestTime.Set || resp.DigestTime.Value != "09:00" {
			t.Errorf("DigestTime = %+v", resp.DigestTime)
		}
		if !resp.FromDate.Set {
			t.Error("FromDate should be set")
		}
		if !resp.ToDate.Set {
			t.Error("ToDate should be set")
		}
	})

	t.Run("success without dates", func(t *testing.T) {
		h := &NotificationHandler{
			notificationService: &mockNotificationService{
				getNotificationConfigFn: func(_ context.Context, _ uint) (*dto.NotificationConfigResponse, error) {
					return &dto.NotificationConfigResponse{DigestTime: "08:00"}, nil
				},
			},
		}

		resp, err := h.GetNotificationConfig(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.FromDate.Set {
			t.Error("expected empty FromDate")
		}
	})

	t.Run("invalid date string silently drops", func(t *testing.T) {
		h := &NotificationHandler{
			notificationService: &mockNotificationService{
				getNotificationConfigFn: func(_ context.Context, _ uint) (*dto.NotificationConfigResponse, error) {
					return &dto.NotificationConfigResponse{
						FromDate:   "not-a-date",
						DigestTime: "08:00",
					}, nil
				},
			},
		}

		resp, err := h.GetNotificationConfig(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.FromDate.Set {
			t.Error("expected FromDate to be not set for invalid date string")
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := &NotificationHandler{
			notificationService: &mockNotificationService{
				getNotificationConfigFn: func(_ context.Context, _ uint) (*dto.NotificationConfigResponse, error) {
					return nil, errors.New("some error")
				},
			},
		}

		_, err := h.GetNotificationConfig(t.Context())
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d", statusErr.StatusCode)
		}
	})
}

func TestNotificationHandler_UpdateNotificationConfig(t *testing.T) {
	t.Run("success with all fields", func(t *testing.T) {
		var capturedReq *dto.NotificationConfigRequest

		h := &NotificationHandler{
			notificationService: &mockNotificationService{
				updateNotificationConfigFn: func(_ context.Context, _ uint, req *dto.NotificationConfigRequest) error {
					capturedReq = req
					return nil
				},
			},
		}

		req := &api.NotificationConfig{
			FromDate:   api.NewOptDate(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			ToDate:     api.NewOptDate(time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)),
			DigestTime: api.NewOptString("08:00"),
		}
		err := h.UpdateNotificationConfig(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedReq == nil {
			t.Fatal("expected captured request")
		}
		if capturedReq.FromDate != "2026-06-01" {
			t.Errorf("FromDate = %q", capturedReq.FromDate)
		}
		if capturedReq.DigestTime != "08:00" {
			t.Errorf("DigestTime = %q", capturedReq.DigestTime)
		}
	})

	t.Run("success with only digest time", func(t *testing.T) {
		h := &NotificationHandler{
			notificationService: &mockNotificationService{
				updateNotificationConfigFn: func(_ context.Context, _ uint, req *dto.NotificationConfigRequest) error {
					if req.FromDate != "" || req.ToDate != "" {
						t.Error("expected empty dates")
					}
					return nil
				},
			},
		}

		req := &api.NotificationConfig{
			DigestTime: api.NewOptString("09:00"),
		}
		err := h.UpdateNotificationConfig(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := &NotificationHandler{
			notificationService: &mockNotificationService{
				updateNotificationConfigFn: func(_ context.Context, _ uint, _ *dto.NotificationConfigRequest) error {
					return apperrors.ErrBadRequest
				},
			},
		}

		req := &api.NotificationConfig{DigestTime: api.NewOptString("08:00")}
		err := h.UpdateNotificationConfig(t.Context(), req)
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d", statusErr.StatusCode)
		}
	})
}

func TestNotificationHandler_SendReport(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		sent := false
		h := &NotificationHandler{
			notificationService: &mockNotificationService{
				sendReportFn: func(_ context.Context, _ uint) error {
					sent = true
					return nil
				},
			},
		}

		err := h.SendReport(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !sent {
			t.Error("SendReport not forwarded")
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := &NotificationHandler{
			notificationService: &mockNotificationService{
				sendReportFn: func(_ context.Context, _ uint) error {
					return errors.New("service error")
				},
			},
		}

		err := h.SendReport(t.Context())
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d", statusErr.StatusCode)
		}
	})
}
