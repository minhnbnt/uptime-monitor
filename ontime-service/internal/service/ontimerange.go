package service

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
)

type OntineRangeRepository interface {
	BatchGetOntimeRange(ctx context.Context, req []ontimerepo.BatchGetOntimeRangeRequest) ([]ontimerepo.RangeEvent, error)
}

type OntimeRangeService struct {
	repo   OntineRangeRepository
	logger *slog.Logger
}

func NewOntimeRangeService(repo OntineRangeRepository, l *slog.Logger) *OntimeRangeService {
	return &OntimeRangeService{repo: repo, logger: l}
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

	events, err := s.repo.BatchGetOntimeRange(ctx, []ontimerepo.BatchGetOntimeRangeRequest{
		{ServerID: serverID, From: from, To: to},
	})

	if err != nil {
		s.logger.Error("failed to get events for range", slog.Any("error", err))
		return nil, err
	}

	uptime := CalculateRangeOntime(events, from, to)
	intervals := CalculateIntervals(events, from, to, resolution)

	totalSeconds := to.Sub(from).Seconds()
	onlineSeconds := uptime / 100 * totalSeconds

	return &dto.UptimeResponse{
		ServerID:      serverID,
		Uptime:        uptime,
		From:          from.Format(time.RFC3339),
		To:            to.Format(time.RFC3339),
		TotalSeconds:  totalSeconds,
		OnlineSeconds: onlineSeconds,
		Intervals:     intervals,
	}, nil
}

func convertRangeToRawEvents(events []ontimerepo.RangeEvent, from time.Time) []ontimerepo.RawEvent {

	if len(events) == 0 {
		return nil
	}

	raw := make([]ontimerepo.RawEvent, 0, len(events)+1)

	startStatus := ""
	idx := sort.Search(len(events), func(i int) bool {
		return !events[i].Time.Before(from)
	})

	for i := idx - 1; i >= 0; i-- {
		if events[i].Status != "" {
			startStatus = events[i].Status
			break
		}
	}

	if startStatus == "" {
		startStatus = events[0].StartStatus
	}

	firstEventIsAfter := events[0].Time.After(from)
	if startStatus != "" && firstEventIsAfter {
		raw = append(raw, ontimerepo.RawEvent{
			ServerID: events[0].ServerID,
			Day:      utils.TruncateDay(from),
			Status:   startStatus,
			Time:     from,
		})
	}

	for _, e := range events {

		if e.Status == "" {
			continue
		}

		raw = append(raw, ontimerepo.RawEvent{
			ServerID: e.ServerID,
			Day:      utils.TruncateDay(e.Time),
			Status:   e.Status,
			Time:     e.Time,
		})
	}

	return raw
}

func CalculateRangeOntime(events []ontimerepo.RangeEvent, from, to time.Time) float64 {

	if len(events) == 0 {
		return 0
	}

	return CalculateOntime(convertRangeToRawEvents(events, from), from, to)
}

func CalculateIntervals(events []ontimerepo.RangeEvent, from, to time.Time, resolution time.Duration) []dto.IntervalResult {

	intervals := utils.SplitIntervals(from, to, resolution)

	return lo.Map(intervals, func(iv [2]time.Time, _ int) dto.IntervalResult {

		uptime := CalculateRangeOntime(events, iv[0], iv[1])

		return dto.IntervalResult{
			From:   iv[0].Format(time.RFC3339),
			To:     iv[1].Format(time.RFC3339),
			Uptime: uptime,
		}
	})
}
