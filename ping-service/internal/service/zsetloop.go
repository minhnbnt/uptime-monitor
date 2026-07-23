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

type ZsetLoopService struct {
	logger           *slog.Logger
	schedulerStorage *scheduler.ZSetScheduleRepository
	endpointProvider endpointProvider
}

func RegisterLoopService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ZsetLoopService, error) {
		return &ZsetLoopService{
			logger:           do.MustInvoke[*slog.Logger](i),
			schedulerStorage: do.MustInvoke[*scheduler.ZSetScheduleRepository](i),
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

func (s *ZsetLoopService) runIteration(ctx context.Context, due []scheduler.ScheduledTask, dueHandler DueHandler) error {

	ids := lo.Map(due, func(task scheduler.ScheduledTask, _ int) uint { return task.EndpointID })
	endpointMap, err := s.endpointProvider.GetBatch(ctx, ids)
	if err != nil {
		return err
	}

	endpoints := maps.Values(endpointMap)
	dueHandler(ctx, endpoints)

	return nil
}

func (s *ZsetLoopService) Run(ctx context.Context, shardID uint, claimLimit int64, dueHandler DueHandler) {

	for ctx.Err() == nil {

		due, next, hasNext, err := s.schedulerStorage.ClaimDueTasksForShard(ctx, shardID, claimLimit)
		if err != nil {
			s.logger.Error("failed to claim due tasks", slog.Any("error", err))
			sleepCtx(ctx, defaultSleepDuration)
			continue
		}

		due, err = s.schedulerStorage.MoveIfWrongShard(ctx, shardID, due)
		if err != nil {
			s.logger.Error("failed to move wrong-shard tasks", slog.Any("error", err))
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
