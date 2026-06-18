package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"

	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	authrepo "github.com/minhnbnt/uptime-monitor/internal/features/auth/repository"
	infra "github.com/minhnbnt/uptime-monitor/internal/features/ping/infrastructure"
	pingrepo "github.com/minhnbnt/uptime-monitor/internal/features/ping/repository"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

const thirtyDays = 30 * 24 * time.Hour
const maxDigestRange = thirtyDays

type DigestService struct {
	eventRepo  EventRepository
	userRepo   UserRepository
	configRepo NotificationConfigRepository
	mailer     MailSender
	logger     logger.Logger
}

func RegisterDigestService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*DigestService, error) {
		return &DigestService{
			eventRepo:  do.MustInvoke[*pingrepo.ServerEventRepository](i),
			userRepo:   do.MustInvoke[*authrepo.UserRepository](i),
			configRepo: do.MustInvoke[*pingrepo.NotificationConfigRepository](i),
			mailer:     do.MustInvoke[*infra.Mailer](i),
			logger:     do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (s *DigestService) SendUserDigest(ctx context.Context, userID uint) error {

	cfg, err := s.configRepo.GetByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get notification config", logger.Error(err))
		return apperrors.ErrInternal
	}

	if cfg == nil || !cfg.Active {
		return fmt.Errorf("no active config for user %d", userID)
	}

	return s.SendReport(ctx, userID, cfg.FromDate)
}

func (s *DigestService) SendReport(ctx context.Context, userID uint, from time.Time) error {

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to find user", logger.Error(err))
		return apperrors.ErrInternal
	}
	if user == nil {
		s.logger.Error("user not found", logger.Int("user_id", int(userID)))
		return fmt.Errorf("user %d: %w", userID, apperrors.ErrNotFound)
	}

	now := time.Now()
	if now.Sub(from) > maxDigestRange {
		from = now.Add(-maxDigestRange)
	}

	events, err := s.eventRepo.GetEnrichedEventsByUser(ctx, userID, from, now)
	if err != nil {
		s.logger.Error("failed to get enriched events", logger.Error(err))
		return apperrors.ErrInternal
	}

	rows := make([]infra.ReportRow, len(events))
	for i, e := range events {
		rows[i] = infra.ReportRow{
			ServerName: e.ServerName,
			URL:        e.URL,
			Status:     e.Status,
			Time:       e.Time,
		}
	}

	excelBytes, err := infra.GenerateStatusReport(rows)
	if err != nil {
		s.logger.Error("failed to generate excel report", logger.Error(err))
		return apperrors.ErrInternal
	}

	subject := fmt.Sprintf("Uptime Monitor - Daily Digest - %s", now.Format("2006-01-02"))
	if err := s.mailer.Send(user.Email, subject, excelBytes); err != nil {
		s.logger.Error("failed to send mail", logger.Error(err))
		return apperrors.ErrInternal
	}

	return nil
}

var (
	_ EventRepository              = (*pingrepo.ServerEventRepository)(nil)
	_ UserRepository               = (*authrepo.UserRepository)(nil)
	_ MailSender                   = (*infra.Mailer)(nil)
	_ NotificationConfigRepository = (*pingrepo.NotificationConfigRepository)(nil)
)
