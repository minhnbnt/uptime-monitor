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

	// Do not calculate into the future: clamp the range end (and start) to the
	// current time so a `to` that exceeds now is capped at the present.
	now := time.Now()
	if to.After(now) {
		to = now
	}
	if from.After(now) {
		from = now
	}

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
	intervalResults = mergeIntervals(intervalResults)

	return &dto.UptimeResponse{
		ServerID:      serverID,
		Uptime:        uptime.Uptime,
		HasData:       uptime.HasData,
		Partial:       uptime.Partial,
		From:          from.Format(time.RFC3339),
		To:            to.Format(time.RFC3339),
		TotalSeconds:  uptime.TotalSeconds,
		OnlineSeconds: uptime.OnlineSeconds,
		Intervals:     intervalResults,
	}, nil
}

func mergeIntervals(intervals []dto.IntervalResult) []dto.IntervalResult {

	if len(intervals) <= 1 {
		return intervals
	}

	merged := []dto.IntervalResult{intervals[0]}

	for i := 1; i < len(intervals); i++ {

		last, cur := &merged[len(merged)-1], intervals[i]
		sameBucket := last.HasData == cur.HasData && (!cur.HasData || last.Uptime == cur.Uptime)

		if last.To == cur.From && sameBucket {
			last.To = cur.To
		} else {
			merged = append(merged, cur)
		}
	}

	return merged
}

func (o OntimeCalculator) CalculateIntervals(events []ontimerepo.Event, intervals []utils.Interval) []dto.IntervalResult {
	return lo.Map(intervals, func(iv utils.Interval, _ int) dto.IntervalResult {
		result := o.CalculateOntime(events, iv.Start, iv.End)
		return dto.IntervalResult{
			From:    iv.Start.Format(time.RFC3339),
			To:      iv.End.Format(time.RFC3339),
			Uptime:  result.Uptime,
			HasData: result.HasData,
		}
	})
}
