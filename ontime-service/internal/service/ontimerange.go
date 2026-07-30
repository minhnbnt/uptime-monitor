package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
)

type OntineRangeRepository interface {
	BatchGetOntimeRange(ctx context.Context, req []ontimerepo.BatchGetOntimeRangeRequest) ([]ontimerepo.ServerEvent, error)
}

type OntimeRangeService struct {
	repo   OntineRangeRepository
	calc   OntimeCalculator
	logger *slog.Logger
}

func NewOntimeRangeService(repo OntineRangeRepository, l *slog.Logger) *OntimeRangeService {
	return &OntimeRangeService{repo: repo, calc: OntimeCalculator{}, logger: l}
}

func RegisterOntimeRangeService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeRangeService, error) {
		return NewOntimeRangeService(
			do.MustInvoke[*ontimerepo.OntineRepository](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

func (s *OntimeRangeService) CalculateUptime(
	ctx context.Context, serverID uint, from, to time.Time, resolution time.Duration,
) (*dto.UptimeResponse, error) {

	request := []ontimerepo.BatchGetOntimeRangeRequest{
		{ServerID: serverID, From: from, To: to},
	}

	serverEvents, err := s.repo.BatchGetOntimeRange(ctx, request)

	if err != nil {
		s.logger.Error("failed to get events for range", slog.Any("error", err))
		return nil, err
	}

	events := lo.Map(serverEvents, func(se ontimerepo.ServerEvent, _ int) ontimerepo.Event {
		return se.Event
	})

	uptime := s.calc.CalculateOntime(events, from, to)

	intervals := utils.SplitIntervals(from, to, resolution)
	intervalResults := s.calc.CalculateIntervals(events, intervals)

	totalSeconds := to.Sub(from).Seconds()
	onlineSeconds := uptime / 100 * totalSeconds

	return &dto.UptimeResponse{
		ServerID:      serverID,
		Uptime:        uptime,
		From:          from.Format(time.RFC3339),
		To:            to.Format(time.RFC3339),
		TotalSeconds:  totalSeconds,
		OnlineSeconds: onlineSeconds,
		Intervals:     intervalResults,
	}, nil
}

func (o OntimeCalculator) CalculateIntervals(events []ontimerepo.Event, intervals [][2]time.Time) []dto.IntervalResult {
	return lo.Map(intervals, func(iv [2]time.Time, _ int) dto.IntervalResult {
		return dto.IntervalResult{
			From:   iv[0].Format(time.RFC3339),
			To:     iv[1].Format(time.RFC3339),
			Uptime: o.CalculateOntime(events, iv[0], iv[1]),
		}
	})
}
