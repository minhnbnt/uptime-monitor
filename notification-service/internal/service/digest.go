package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure/excelgen"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure/ontimeclient"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure/serverclient"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure/userclient"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure/utils"
)

type MailSender interface {
	Send(to, subject string, attachment io.Reader) error
}

type UserAdapter interface {
	FindByID(ctx context.Context, id uint) (*domain.User, error)
}

type ServerAdapter interface {
	List(ctx context.Context, createdByID uint, limit, offset int) ([]domain.Server, error)
	CountByStatus(ctx context.Context, createdByID uint) (total, online, offline int64, err error)
}

type OntimeAdapter interface {
	GetServersOntimeForDates(ctx context.Context, userID uint, servers []domain.Server, dates []time.Time) (map[uint][]domain.OntimeStats, error)
}

func (s *DigestService) buildReport(servers []domain.Server, ontimeMap map[uint][]domain.OntimeStats) []excelgen.ServerRow {

	slices.SortFunc(servers, func(a, b domain.Server) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})

	rows := make([]excelgen.ServerRow, 0, len(servers))
	for _, sv := range servers {

		stats := make(map[time.Time]float64)

		for _, stat := range ontimeMap[sv.ID] {
			stats[utils.TruncateDay(stat.Date)] = stat.Stats
		}

		rows = append(rows, excelgen.ServerRow{
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
	userRepo   UserAdapter
	serverRepo ServerAdapter
	ontimeSvc  OntimeAdapter
	mailer     MailSender
	logger     *slog.Logger
}

func RegisterDigestService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*DigestService, error) {
		return &DigestService{
			configRepo: do.MustInvoke[*repository.NotificationConfigRepository](i),
			serverRepo: do.MustInvoke[*serverclient.Client](i),
			userRepo:   do.MustInvoke[*userclient.Client](i),
			ontimeSvc:  do.MustInvoke[*ontimeclient.Client](i),
			mailer:     do.MustInvoke[*infrastructure.Mailer](i),
			logger:     do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (s *DigestService) SendUserDigest(ctx context.Context, userID uint) error {

	s.logger.Info("SendUserDigest: start", slog.Uint64("user_id", uint64(userID)))

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {

		s.logger.Error(
			"SendUserDigest: failed to find user",
			slog.Uint64("user_id", uint64(userID)),
			slog.Any("error", err),
		)

		return apperrors.ErrInternal
	}
	if user == nil {

		s.logger.Warn(
			"SendUserDigest: user not found, skipping",
			slog.Uint64("user_id", uint64(userID)),
		)

		return nil
	}

	cfg, err := s.configRepo.GetByUserID(ctx, userID)
	if err != nil {

		s.logger.Error(
			"SendUserDigest: failed to get notification config",
			slog.Uint64("user_id", uint64(userID)),
			slog.Any("error", err),
		)

		return apperrors.ErrInternal
	}

	if cfg == nil || !cfg.Active {

		s.logger.Info(
			"SendUserDigest: no active config, skipping",
			slog.Uint64("user_id", uint64(userID)),
		)

		return nil
	}

	s.logger.Debug(
		"SendUserDigest: config active",
		slog.Uint64("user_id", uint64(userID)),
		slog.String("from_date", cfg.FromDate.Format("2006-01-02")),
	)

	return s.SendReport(ctx, userID, cfg.FromDate)
}

func (s *DigestService) SendReport(ctx context.Context, userID uint, from time.Time) error {

	s.logger.Info(
		"SendReport: start",
		slog.Uint64("user_id", uint64(userID)),
		slog.String("from_date", from.Format("2006-01-02")),
	)

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {

		s.logger.Error(
			"SendReport: failed to find user",
			slog.Uint64("user_id", uint64(userID)),
			slog.Any("error", err),
		)

		return apperrors.ErrInternal
	}
	if user == nil {

		s.logger.Error(
			"SendReport: user not found",
			slog.Uint64("user_id", uint64(userID)),
		)

		return fmt.Errorf("user %d: %w", userID, apperrors.ErrNotFound)
	}

	now := time.Now()
	if now.Sub(from) > maxDigestRange {
		from = now.Add(-maxDigestRange)
	}

	servers, err := s.serverRepo.List(ctx, userID, maxReportServers, 0)
	if err != nil {

		s.logger.Error(
			"SendReport: failed to list servers",
			slog.Uint64("user_id", uint64(userID)),
			slog.Any("error", err),
		)

		return apperrors.ErrInternal
	}

	s.logger.Debug(
		"SendReport: listed servers",
		slog.Uint64("user_id", uint64(userID)),
		slog.Int("count", len(servers)),
	)

	dates := utils.BuildDateRange(from, now)

	ontimeMap, err := s.ontimeSvc.GetServersOntimeForDates(ctx, userID, servers, dates)
	if err != nil {

		s.logger.Error(
			"SendReport: failed to get ontime stats",
			slog.Uint64("user_id", uint64(userID)),
			slog.Any("error", err),
		)

		return apperrors.ErrInternal
	}

	s.logger.Debug(
		"SendReport: got ontime stats",
		slog.Uint64("user_id", uint64(userID)),
		slog.Int("servers_with_stats", len(ontimeMap)),
	)

	rows := s.buildReport(servers, ontimeMap)

	total, online, offline, err := s.serverRepo.CountByStatus(ctx, userID)
	if err != nil {

		s.logger.Error(
			"SendReport: failed to count servers by status",
			slog.Uint64("user_id", uint64(userID)),
			slog.Any("error", err),
		)

		return apperrors.ErrInternal
	}

	s.logger.Debug(
		"SendReport: server counts",
		slog.Uint64("user_id", uint64(userID)),
		slog.Int64("total", total),
		slog.Int64("online", online),
		slog.Int64("offline", offline),
	)

	summary := excelgen.ServerSummary{Total: total, Online: online, Offline: offline}

	reader, err := excelgen.GenerateStatusReport(rows, &summary)
	if err != nil {

		s.logger.Error(
			"SendReport: failed to generate excel report",
			slog.Uint64("user_id", uint64(userID)),
			slog.Any("error", err),
		)

		return apperrors.ErrInternal
	}

	defer reader.Close()

	subject := fmt.Sprintf("Uptime Monitor - Daily Digest - %s", now.Format("2006-01-02"))
	if err := s.mailer.Send(user.Email, subject, reader); err != nil {

		s.logger.Error(
			"SendReport: failed to send mail",
			slog.Uint64("user_id", uint64(userID)),
			slog.String("email", user.Email),
			slog.Any("error", err),
		)

		return apperrors.ErrInternal
	}

	s.logger.Info(
		"SendReport: digest sent",
		slog.Uint64("user_id", uint64(userID)),
		slog.String("email", user.Email),
	)

	return nil
}
