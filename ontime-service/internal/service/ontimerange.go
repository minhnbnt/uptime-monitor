package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
)

type UptimeRepository interface {
	BatchGetUptime(ctx context.Context, req []repository.BatchGetOntimeRequest) ([]repository.UptimeRow, error)
}

type OntimeRangeService struct {
	uptimeRepo UptimeRepository
	ownerRepo  ServerOwnerRepository
	logger     *slog.Logger
}

func RegisterOntimeRangeService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeRangeService, error) {
		return &OntimeRangeService{
			uptimeRepo: do.MustInvoke[*repository.OntimeUptimeRepository](i),
			ownerRepo:  do.MustInvoke[*repository.ServerOwnerRepository](i),
			logger:     do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (s *OntimeRangeService) CalculateUptime(
	ctx context.Context, in dto.CalculateUptimeInput,
) (*dto.UptimeResponse, error) {

	serverID := in.ServerID

	owned, err := s.ownerRepo.GetOwnedServers(ctx, in.UserID, []uint{serverID})
	if err != nil {
		return nil, err
	}
	if len(owned) == 0 {
		return nil, apperrors.ErrForbidden
	}

	from, to := in.From, in.To
	now := time.Now()
	if to.After(now) {
		to = now
	}

	if from.After(now) {
		from = now
	}

	from = from.Truncate(time.Microsecond)
	to = to.Truncate(time.Microsecond)

	if err := validateRange(from, to); err != nil {
		return nil, err
	}

	resolution := in.Resolution

	request := []repository.BatchGetOntimeRequest{
		{EndpointID: serverID, From: from, To: to},
	}

	rows, err := s.uptimeRepo.BatchGetUptime(ctx, request)
	if err != nil {
		s.logger.Error("failed to get uptime for range", slog.Any("error", err))
		return nil, err
	}

	overall := repository.UptimeRow{
		EndpointID:     serverID,
		From:           from,
		To:             to,
		ObservedFrom:   from,
		ObservedTo:     to,
		HasData:        false,
		OnlineSeconds:  0,
		UnknownSeconds: 0,
	}

	if len(rows) > 0 {
		overall = rows[0]
	}

	intervals := utils.SplitIntervals(from, to, resolution)
	intervalResults := s.calculateIntervals(ctx, serverID, intervals)
	intervalResults = mergeIntervals(intervalResults)

	return &dto.UptimeResponse{
		ServerID:      serverID,
		From:          from.Format(time.RFC3339),
		To:            to.Format(time.RFC3339),
		Uptime:        overall.UptimePercent(),
		HasData:       overall.HasData,
		Partial:       overall.ObservedFrom.After(from),
		TotalSeconds:  overall.TotalSeconds(),
		OnlineSeconds: overall.OnlineSeconds,
		Intervals:     intervalResults,
	}, nil
}

func (s *OntimeRangeService) calculateIntervals(
	ctx context.Context, serverID uint, intervals []utils.Interval,
) []dto.IntervalResult {

	if len(intervals) == 0 {
		return nil
	}

	requests := lo.Map(intervals, func(iv utils.Interval, _ int) repository.BatchGetOntimeRequest {
		return repository.BatchGetOntimeRequest{
			EndpointID: serverID,
			To:         iv.End.Truncate(time.Microsecond).UTC(),
			From:       iv.Start.Truncate(time.Microsecond).UTC(),
		}
	})

	rows, err := s.uptimeRepo.BatchGetUptime(ctx, requests)
	if err != nil {
		s.logger.Error("failed to get interval uptimes", slog.Any("error", err))
		return nil
	}

	rowMap := make(map[time.Time]repository.UptimeRow, len(rows))
	for _, row := range rows {
		key := row.From.Truncate(time.Microsecond).UTC()
		rowMap[key] = row
	}

	return lo.Map(intervals, func(iv utils.Interval, _ int) dto.IntervalResult {

		result := dto.IntervalResult{
			From:    iv.Start.Format(time.RFC3339),
			To:      iv.End.Format(time.RFC3339),
			HasData: false,
			Uptime:  0,
		}

		if row, ok := rowMap[iv.Start]; ok {
			result.Uptime = row.UptimePercent()
			result.HasData = row.HasData
		}

		return result
	})
}

func validateRange(from, to time.Time) error {

	if from.After(to) || from.Equal(to) {
		return fmt.Errorf("%w: from must be before to", apperrors.ErrBadRequest)
	}

	if to.Sub(from) > 90*24*time.Hour {
		return fmt.Errorf("%w: range must not exceed 90 days", apperrors.ErrBadRequest)
	}

	return nil
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

var _ UptimeRepository = (*repository.OntimeUptimeRepository)(nil)
