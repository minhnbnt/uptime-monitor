package service

import (
	"context"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	notificationrepo "github.com/minhnbnt/uptime-monitor/internal/repository/notification"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/temporal"
)

const dateLayout = "2006-01-02"

type NotificationService struct {
	configRepo    NotificationConfigRepository
	digestStarter DigestStarter
	logger        logger.Logger
}

func RegisterNotificationService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*NotificationService, error) {
		return &NotificationService{
			configRepo:    do.MustInvoke[*notificationrepo.NotificationConfigRepository](i),
			digestStarter: do.MustInvoke[*temporal.DigestStarter](i),
			logger:        do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (s *NotificationService) GetNotificationConfig(ctx context.Context, userID uint) (*dto.NotificationConfigResponse, error) {
	cfg, err := s.configRepo.GetByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get notification config", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	if cfg == nil {
		return &dto.NotificationConfigResponse{
			DigestTime: "08:00",
		}, nil
	}

	return &dto.NotificationConfigResponse{
		FromDate:   cfg.FromDate.Format(dateLayout),
		ToDate:     cfg.ToDate.Format(dateLayout),
		DigestTime: cfg.DigestTime,
	}, nil
}

func (s *NotificationService) UpdateNotificationConfig(ctx context.Context, userID uint, req *dto.NotificationConfigRequest) error {

	cfg := &domain.NotificationConfig{
		UserID:     userID,
		Active:     req.FromDate != "" && req.ToDate != "",
		DigestTime: req.DigestTime,
	}

	if req.FromDate != "" {
		fromDate, err := time.Parse(dateLayout, req.FromDate)
		if err != nil {
			return apperrors.ErrBadRequest
		}
		cfg.FromDate = fromDate
	}

	if req.ToDate != "" {
		toDate, err := time.Parse(dateLayout, req.ToDate)
		if err != nil {
			return apperrors.ErrBadRequest
		}
		cfg.ToDate = toDate
	}

	if err := s.configRepo.Upsert(ctx, cfg); err != nil {
		s.logger.Error("failed to update notification config", logger.Error(err))
		return apperrors.ErrInternal
	}

	if cfg.Active {
		if err := s.digestStarter.UpsertSchedule(ctx, userID, cfg.FromDate, cfg.ToDate, cfg.DigestTime); err != nil {
			s.logger.Error("failed to upsert digest schedule", logger.Error(err))
			return apperrors.ErrInternal
		}
	} else {
		if err := s.digestStarter.DeleteSchedule(ctx, userID); err != nil {
			s.logger.Error("failed to delete digest schedule", logger.Error(err))
		}
	}

	return nil
}

func (s *NotificationService) SendReport(ctx context.Context, userID uint) error {

	if err := s.digestStarter.StartDigest(ctx, userID); err != nil {
		s.logger.Error("failed to start digest workflow", logger.Error(err))
		return apperrors.ErrInternal
	}

	return nil
}
