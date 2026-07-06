package service

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	authrepo "github.com/minhnbnt/uptime-monitor/internal/features/auth/repository"
	digestinfra "github.com/minhnbnt/uptime-monitor/internal/features/digest/infrastructure"
	digestrepo "github.com/minhnbnt/uptime-monitor/internal/features/digest/repository"
	ontimedto "github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
	ontimesvc "github.com/minhnbnt/uptime-monitor/internal/features/ontime/service"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

func (s *DigestService) buildReport(servers []domain.Server, ontimeMap map[uint][]ontimedto.OntimeStats) []digestinfra.ServerRow {

	slices.SortFunc(servers, func(a, b domain.Server) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})

	rows := make([]digestinfra.ServerRow, 0, len(servers))
	for _, sv := range servers {

		stats := make(map[time.Time]float64)
		for _, stat := range ontimeMap[sv.ID] {
			stats[utils.TruncateDay(stat.Date)] = stat.Stats
		}

		rows = append(rows, digestinfra.ServerRow{
			ServerID:   sv.ID,
			ServerName: sv.Name,
			Stats:      stats,
		})
	}

	return rows
}

const (
	thirtyDays       = 30 * 24 * time.Hour
	maxDigestRange   = thirtyDays
	maxReportServers = 10000
)

type DigestService struct {
	configRepo NotificationConfigRepository
	userRepo   UserRepository
	serverRepo ServerLister
	ontimeSvc  OntimeStatsService
	mailer     MailSender
	logger     logger.Logger
}

func RegisterDigestService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*DigestService, error) {
		return &DigestService{
			configRepo: do.MustInvoke[*digestrepo.NotificationConfigRepository](i),
			serverRepo: do.MustInvoke[*serverrepo.ServerRepository](i),
			userRepo:   do.MustInvoke[*authrepo.UserRepository](i),
			ontimeSvc:  do.MustInvoke[*ontimesvc.OntimeService](i),
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

	servers, err := s.serverRepo.List(ctx, userID, maxReportServers, 0)
	if err != nil {
		s.logger.Error("failed to list servers", logger.Error(err))
		return apperrors.ErrInternal
	}

	dates := utils.BuildDateRange(from, now)

	ontimeMap, err := s.ontimeSvc.GetServersOntimeForDates(ctx, servers, dates)
	if err != nil {
		s.logger.Error("failed to get ontime stats", logger.Error(err))
		return apperrors.ErrInternal
	}

	rows := s.buildReport(servers, ontimeMap)

	total, online, offline, err := s.serverRepo.CountByStatus(ctx, userID)
	if err != nil {
		s.logger.Error("failed to count servers by status", logger.Error(err))
		return apperrors.ErrInternal
	}

	summary := digestinfra.ServerSummary{Total: total, Online: online, Offline: offline}

	reader, err := digestinfra.GenerateStatusReport(rows, &summary)
	if err != nil {
		s.logger.Error("failed to generate excel report", logger.Error(err))
		return apperrors.ErrInternal
	}

	defer reader.Close()

	subject := fmt.Sprintf("Uptime Monitor - Daily Digest - %s", now.Format("2006-01-02"))
	if err := s.mailer.Send(user.Email, subject, reader); err != nil {
		s.logger.Error("failed to send mail", logger.Error(err))
		return apperrors.ErrInternal
	}

	return nil
}

var (
	_ UserRepository               = (*authrepo.UserRepository)(nil)
	_ MailSender                   = (*digestinfra.Mailer)(nil)
	_ NotificationConfigRepository = (*digestrepo.NotificationConfigRepository)(nil)
)
