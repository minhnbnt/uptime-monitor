package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure"
)

const dateLayout = "2006-01-02"

type DigestStarter interface {
	StartDigest(ctx context.Context, userID uuid.UUID) error
	UpsertSchedule(ctx context.Context, userID uuid.UUID, cfg domain.ScheduleConfig) error
	DeleteSchedule(ctx context.Context, userID uuid.UUID) error
	DescribeSchedule(ctx context.Context, userID uuid.UUID) (*domain.ScheduleInfo, error)
}

type NotificationService struct {
	digestStarter DigestStarter
	logger        *slog.Logger
}

func RegisterNotificationService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*NotificationService, error) {
		return &NotificationService{
			digestStarter: do.MustInvoke[*infrastructure.TemporalDigestStarter](i),
			logger:        do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (s *NotificationService) GetNotificationConfig(ctx context.Context, userID uuid.UUID) (*dto.NotificationConfigResponse, error) {

	info, err := s.digestStarter.DescribeSchedule(ctx, userID)
	if err != nil {
		s.logger.Error("failed to describe schedule", slog.Any("error", err))
		return nil, fmt.Errorf("describe schedule: %w", apperrors.ErrInternal)
	}

	if !info.Exists {
		return nil, fmt.Errorf("notification config not found: %w", apperrors.ErrNotFound)
	}

	resp := &dto.NotificationConfigResponse{
		DigestTime: info.DigestTime,
	}

	if !info.FromDate.IsZero() {
		resp.FromDate = info.FromDate.Format(dateLayout)
	}
	if !info.ToDate.IsZero() {
		resp.ToDate = info.ToDate.Format(dateLayout)
	}

	return resp, nil
}

func (s *NotificationService) UpdateNotificationConfig(ctx context.Context, userID uuid.UUID, req *dto.NotificationConfigRequest) error {

	active := req.FromDate != "" && req.ToDate != ""

	if active {
		fromDate, err := time.Parse(dateLayout, req.FromDate)
		if err != nil {
			return fmt.Errorf("parse from_date: %w", apperrors.ErrBadRequest)
		}

		toDate, err := time.Parse(dateLayout, req.ToDate)
		if err != nil {
			return fmt.Errorf("parse to_date: %w", apperrors.ErrBadRequest)
		}

		config := domain.ScheduleConfig{
			FromDate:   fromDate,
			ToDate:     toDate,
			DigestTime: req.DigestTime,
		}

		if err := s.digestStarter.UpsertSchedule(ctx, userID, config); err != nil {
			s.logger.Error("failed to upsert digest schedule", slog.Any("error", err))
			return fmt.Errorf("upsert schedule: %w", apperrors.ErrInternal)
		}

	} else {
		if err := s.digestStarter.DeleteSchedule(ctx, userID); err != nil {
			s.logger.Error("failed to delete digest schedule", slog.Any("error", err))
			return fmt.Errorf("delete schedule: %w", apperrors.ErrInternal)
		}
	}

	return nil
}

func (s *NotificationService) SendReport(ctx context.Context, userID uuid.UUID) error {

	info, err := s.digestStarter.DescribeSchedule(ctx, userID)
	if err != nil {
		s.logger.Error("failed to describe schedule", slog.Any("error", err))
		return fmt.Errorf("describe schedule: %w", apperrors.ErrInternal)
	}
	if !info.Exists {
		return fmt.Errorf("notification config not found: %w", apperrors.ErrNotFound)
	}

	if err := s.digestStarter.StartDigest(ctx, userID); err != nil {
		s.logger.Error("failed to start digest workflow", slog.Any("error", err))
		return fmt.Errorf("start digest: %w", apperrors.ErrInternal)
	}

	return nil
}
