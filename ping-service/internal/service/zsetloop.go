package service

import (
	"context"
	"iter"
	"log/slog"
	"maps"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	scheduler "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
)

const (
	defaultSleepDuration = 5 * time.Second
)

type DueHandler func(ctx context.Context, tasks iter.Seq[*domain.Endpoint])

type endpointProvider interface {
	GetBatch(ctx context.Context, ids []uint) (map[uint]*domain.Endpoint, error)
}

type scoreUpdater interface {
	UpdateBatch(ctx context.Context, items map[uint]int64) error
}

type LoopService struct {
	logger           *slog.Logger
	schedulerStorage *scheduler.ZSetScheduleRepository
	scoreUpdater     scoreUpdater
	endpointProvider endpointProvider
}

func RegisterLoopService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*LoopService, error) {
		return &LoopService{
			logger:           do.MustInvoke[*slog.Logger](i),
			schedulerStorage: do.MustInvoke[*scheduler.ZSetScheduleRepository](i),
			scoreUpdater:     do.MustInvoke[*scheduler.ScoreUpdater](i),
			endpointProvider: do.MustInvoke[*scheduler.EndpointProvider](i),
		}, nil
	})
}

func sleepCtx(ctx context.Context, d time.Duration) {

	if d <= 0 {
		return
	}

	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func getSleepDuration(next scheduler.ScheduledTask, hasNext bool) time.Duration {

	if !hasNext {
		return defaultSleepDuration
	}

	nextTime := time.UnixMilli(next.Score)
	if nextTime.Before(time.Now()) {
		return 0
	}

	return time.Until(nextTime)
}

func calculateNextScore(score int64, interval time.Duration) int64 {

	nowUnixMilli := time.Now().UnixMilli()
	intervalMilliseconds := interval.Milliseconds()

	next := score
	if next <= nowUnixMilli {
		missed := (nowUnixMilli-next)/intervalMilliseconds + 1
		next += missed * intervalMilliseconds
	}

	return next
}

func (s *LoopService) runIteration(ctx context.Context, due []scheduler.ScheduledTask, dueHandler DueHandler) error {

	ids := lo.Map(due, func(task scheduler.ScheduledTask, _ int) uint { return task.EndpointID })
	endpointMap, err := s.endpointProvider.GetBatch(ctx, ids)
	if err != nil {
		return err
	}

	endpoints := maps.Values(endpointMap)
	dueHandler(ctx, endpoints)

	updates := make(map[uint]int64, len(due))
	for _, task := range due {

		ep, ok := endpointMap[task.EndpointID]

		if !ok {
			s.logger.Warn(
				"endpoint not found in batch, skipping reschedule",
				slog.Int("endpoint_id", int(task.EndpointID)),
			)
			continue
		}

		updates[task.EndpointID] = calculateNextScore(task.Score, ep.Interval)
	}

	if len(updates) > 0 {
		return s.scoreUpdater.UpdateBatch(ctx, updates)
	}

	return nil
}

func (s *LoopService) Run(ctx context.Context, shardID uint, claimLimit int64, dueHandler DueHandler) {

	for ctx.Err() == nil {

		due, next, hasNext, err := s.schedulerStorage.ClaimDueTasksForShard(ctx, shardID, claimLimit)
		if err != nil {
			s.logger.Error("failed to claim due tasks", slog.Any("error", err))
			sleepCtx(ctx, defaultSleepDuration)
			continue
		}

		err = s.runIteration(ctx, due, dueHandler)
		if err != nil {
			s.logger.Error("failed to run iteration", slog.Any("error", err))
			sleepCtx(ctx, defaultSleepDuration)
			continue
		}

		if len(due) != int(claimLimit) {
			sleepCtx(ctx, getSleepDuration(next, hasNext))
		}
	}
}
