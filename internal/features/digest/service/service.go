package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	authrepo "github.com/minhnbnt/uptime-monitor/internal/features/auth/repository"
	digestinfra "github.com/minhnbnt/uptime-monitor/internal/features/digest/infrastructure"
	digestrepo "github.com/minhnbnt/uptime-monitor/internal/features/digest/repository"
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
			configRepo: do.MustInvoke[*digestrepo.NotificationConfigRepository](i),
			mailer:     do.MustInvoke[*digestinfra.Mailer](i),
			logger:     do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (s *DigestService) SendUserDigest(ctx context.Context, userID uint) error {

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to find user", logger.Error(err))
		return apperrors.ErrInternal
	}
	if user == nil {
		s.logger.Warn("user not found, skipping digest", logger.Int("user_id", int(userID)))
		return nil
	}

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

	rows := lo.Map(events, func(e pingrepo.EnrichedEvent, _ int) digestinfra.ReportRow {
		return digestinfra.ReportRow{
			ServerName: e.ServerName,
			Status:     e.Status,
			Time:       e.Time,
			URL:        e.URL,
		}
	})

	excelBytes, err := digestinfra.GenerateStatusReport(rows)
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
	_ MailSender                   = (*digestinfra.Mailer)(nil)
	_ NotificationConfigRepository = (*digestrepo.NotificationConfigRepository)(nil)
)
