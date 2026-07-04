package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/notification/dto"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

func TestNotificationService_GetNotificationConfig(t *testing.T) {
	now := time.Now()

	t.Run("returns config with dates", func(t *testing.T) {
		mockRepo := &mockNotificationConfigRepository{
			getByUserIDFn: func(_ context.Context, userID uint) (*domain.NotificationConfig, error) {
				return &domain.NotificationConfig{
					UserID:     userID,
					FromDate:   now,
					ToDate:     now.Add(24 * time.Hour),
					DigestTime: "09:00",
				}, nil
			},
		}
		s := &NotificationService{configRepo: mockRepo, logger: logger.NewMockLogger()}
		resp, err := s.GetNotificationConfig(t.Context(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.DigestTime != "09:00" {
			t.Errorf("DigestTime = %q, want %q", resp.DigestTime, "09:00")
		}
		if resp.FromDate == "" {
			t.Error("expected non-empty FromDate")
		}
	})

	t.Run("returns defaults when config is nil", func(t *testing.T) {
		mockRepo := &mockNotificationConfigRepository{
			getByUserIDFn: func(_ context.Context, _ uint) (*domain.NotificationConfig, error) {
				return nil, nil
			},
		}
		s := &NotificationService{configRepo: mockRepo, logger: logger.NewMockLogger()}
		resp, err := s.GetNotificationConfig(t.Context(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.DigestTime != "08:00" {
			t.Errorf("DigestTime = %q, want %q", resp.DigestTime, "08:00")
		}
		if resp.FromDate != "" {
			t.Errorf("expected empty FromDate for nil config, got %q", resp.FromDate)
		}
	})

	t.Run("repo error returns ErrInternal", func(t *testing.T) {
		mockRepo := &mockNotificationConfigRepository{
			getByUserIDFn: func(_ context.Context, _ uint) (*domain.NotificationConfig, error) {
				return nil, errors.New("db error")
			},
		}
		mockLog := logger.NewMockLogger()
		s := &NotificationService{configRepo: mockRepo, logger: mockLog}
		_, err := s.GetNotificationConfig(t.Context(), 1)
		if !errors.Is(err, apperrors.ErrInternal) {
			t.Errorf("got %v, want %v", err, apperrors.ErrInternal)
		}
		if !mockLog.ErrorCalled {
			t.Error("expected Error log")
		}
	})
}

func TestNotificationService_UpdateNotificationConfig(t *testing.T) {
	mockLog := logger.NewMockLogger()

	t.Run("active config with dates", func(t *testing.T) {
		upsertCalled := false
		scheduleCalled := false

		mockRepo := &mockNotificationConfigRepository{
			upsertFn: func(_ context.Context, cfg *domain.NotificationConfig) error {
				upsertCalled = true
				if !cfg.Active {
					t.Error("expected Active=true")
				}
				if cfg.UserID != 1 {
					t.Errorf("UserID = %d", cfg.UserID)
				}
				return nil
			},
		}
		mockDigest := &mockDigestStarter{
			upsertScheduleFn: func(_ context.Context, userID uint, _, _ time.Time, digestTime string) error {
				scheduleCalled = true
				if digestTime != "08:00" {
					t.Errorf("digestTime = %q", digestTime)
				}
				return nil
			},
		}

		s := &NotificationService{configRepo: mockRepo, digestStarter: mockDigest, logger: mockLog}
		req := &dto.NotificationConfigRequest{
			FromDate:   "2026-06-01",
			ToDate:     "2026-06-30",
			DigestTime: "08:00",
		}
		err := s.UpdateNotificationConfig(t.Context(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !upsertCalled {
			t.Error("Upsert not called")
		}
		if !scheduleCalled {
			t.Error("UpsertSchedule not called")
		}
	})

	t.Run("inactive config (empty dates) deletes schedule", func(t *testing.T) {
		deleteCalled := false

		mockRepo := &mockNotificationConfigRepository{
			upsertFn: func(_ context.Context, cfg *domain.NotificationConfig) error {
				if cfg.Active {
					t.Error("expected Active=false")
				}
				return nil
			},
		}
		mockDigest := &mockDigestStarter{
			deleteScheduleFn: func(_ context.Context, userID uint) error {
				deleteCalled = true
				return nil
			},
		}

		s := &NotificationService{configRepo: mockRepo, digestStarter: mockDigest, logger: mockLog}
		req := &dto.NotificationConfigRequest{DigestTime: "08:00"}
		err := s.UpdateNotificationConfig(t.Context(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !deleteCalled {
			t.Error("DeleteSchedule not called")
		}
	})

	t.Run("invalid date format returns ErrBadRequest", func(t *testing.T) {
		mockRepo := &mockNotificationConfigRepository{}
		s := &NotificationService{configRepo: mockRepo, logger: mockLog}
		req := &dto.NotificationConfigRequest{
			FromDate: "invalid-date",
			ToDate:   "2026-06-30",
		}
		err := s.UpdateNotificationConfig(t.Context(), 1, req)
		if !errors.Is(err, apperrors.ErrBadRequest) {
			t.Errorf("got %v, want %v", err, apperrors.ErrBadRequest)
		}
	})

	t.Run("repo upsert error returns ErrInternal", func(t *testing.T) {
		mockRepo := &mockNotificationConfigRepository{
			upsertFn: func(_ context.Context, _ *domain.NotificationConfig) error {
				return errors.New("db error")
			},
		}
		mockLog := logger.NewMockLogger()
		s := &NotificationService{configRepo: mockRepo, logger: mockLog}
		req := &dto.NotificationConfigRequest{DigestTime: "08:00"}
		err := s.UpdateNotificationConfig(t.Context(), 1, req)
		if !errors.Is(err, apperrors.ErrInternal) {
			t.Errorf("got %v, want %v", err, apperrors.ErrInternal)
		}
		if !mockLog.ErrorCalled {
			t.Error("expected Error log")
		}
	})

	t.Run("upsert schedule error returns ErrInternal", func(t *testing.T) {
		mockRepo := &mockNotificationConfigRepository{
			upsertFn: func(_ context.Context, _ *domain.NotificationConfig) error {
				return nil
			},
		}
		mockDigest := &mockDigestStarter{
			upsertScheduleFn: func(_ context.Context, _ uint, _, _ time.Time, _ string) error {
				return errors.New("temporal error")
			},
		}
		mockLog := logger.NewMockLogger()
		s := &NotificationService{configRepo: mockRepo, digestStarter: mockDigest, logger: mockLog}
		req := &dto.NotificationConfigRequest{
			FromDate:   "2026-06-01",
			ToDate:     "2026-06-30",
			DigestTime: "08:00",
		}
		err := s.UpdateNotificationConfig(t.Context(), 1, req)
		if !errors.Is(err, apperrors.ErrInternal) {
			t.Errorf("got %v, want %v", err, apperrors.ErrInternal)
		}
		if !mockLog.ErrorCalled {
			t.Error("expected Error log")
		}
	})
}

func TestNotificationService_SendReport(t *testing.T) {
	mockLog := logger.NewMockLogger()

	t.Run("success", func(t *testing.T) {
		started := false
		mockDigest := &mockDigestStarter{
			startDigestFn: func(_ context.Context, userID uint) error {
				started = true
				return nil
			},
		}
		s := &NotificationService{digestStarter: mockDigest, logger: mockLog}
		err := s.SendReport(t.Context(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !started {
			t.Error("StartDigest not called")
		}
	})

	t.Run("digest error returns ErrInternal", func(t *testing.T) {
		mockDigest := &mockDigestStarter{
			startDigestFn: func(_ context.Context, _ uint) error {
				return errors.New("temporal error")
			},
		}
		mockLog := logger.NewMockLogger()
		s := &NotificationService{digestStarter: mockDigest, logger: mockLog}
		err := s.SendReport(t.Context(), 1)
		if !errors.Is(err, apperrors.ErrInternal) {
			t.Errorf("got %v, want %v", err, apperrors.ErrInternal)
		}
		if !mockLog.ErrorCalled {
			t.Error("expected Error log")
		}
	})
}
