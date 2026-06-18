package services

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"

	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	infra "github.com/minhnbnt/uptime-monitor/internal/monitor/infrastructure"
	authrepo "github.com/minhnbnt/uptime-monitor/internal/repository/auth"
	monitorrepo "github.com/minhnbnt/uptime-monitor/internal/repository/monitor"
	notificationrepo "github.com/minhnbnt/uptime-monitor/internal/repository/notification"
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
			eventRepo:  do.MustInvoke[*monitorrepo.ServerEventRepository](i),
			userRepo:   do.MustInvoke[*authrepo.UserRepository](i),
			configRepo: do.MustInvoke[*notificationrepo.NotificationConfigRepository](i),
			mailer:     do.MustInvoke[*infra.Mailer](i),
			logger:     do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (s *DigestService) SendUserDigest(ctx context.Context, userID uint) error {

	cfg, err := s.configRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get config: %w", err)
	}

	if cfg == nil || !cfg.Active {
		return fmt.Errorf("no active config for user %d", userID)
	}

	return s.SendReport(ctx, userID, cfg.FromDate)
}

func (s *DigestService) SendReport(ctx context.Context, userID uint, from time.Time) error {

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user %d: %w", userID, apperrors.ErrNotFound)
	}

	now := time.Now()
	if now.Sub(from) > maxDigestRange {
		from = now.Add(-maxDigestRange)
	}

	events, err := s.eventRepo.GetEnrichedEventsByUser(ctx, userID, from, now)
	if err != nil {
		return fmt.Errorf("get events: %w", err)
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
		return fmt.Errorf("generate excel: %w", err)
	}

	subject := fmt.Sprintf("Uptime Monitor - Daily Digest - %s", now.Format("2006-01-02"))
	if err := s.mailer.Send(user.Email, subject, excelBytes); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}

	return nil
}
