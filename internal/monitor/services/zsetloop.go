package services

import (
	"context"
	"iter"
	"maps"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	scheduler "github.com/minhnbnt/uptime-monitor/internal/repository/scheduler"
)

const (
	defaultClaimLimit    = 50
	defaultSleepDuration = 5 * time.Second
)

type DueHandler func(ctx context.Context, tasks iter.Seq[*domain.Endpoint])

type LoopService struct {
	logger           logger.Logger
	schedulerStorage *scheduler.ZSetScheduleRepository
	scoreUpdater     *scheduler.ScoreUpdater
	endpointProvider *scheduler.EndpointProvider
}

func RegisterLoopService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*LoopService, error) {
		return &LoopService{
			logger:           do.MustInvoke[logger.Logger](i),
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

func (s *LoopService) runIteration(ctx context.Context, due []scheduler.ScheduledTask, dueHandler DueHandler) error {

	ids := lo.Map(due, func(task scheduler.ScheduledTask, _ int) uint { return task.EndpointID })
	endpointMap, err := s.endpointProvider.GetBatch(ctx, ids)
	if err != nil {
		return err
	}

	endpoints := maps.Values(endpointMap)
	dueHandler(ctx, endpoints)

	updates := lo.SliceToMap(due, func(task scheduler.ScheduledTask) (uint, int64) {
		return task.EndpointID, task.Score + endpointMap[task.EndpointID].Interval.Milliseconds()
	})

	if len(updates) > 0 {
		return s.scoreUpdater.UpdateBatch(ctx, updates)
	}

	return nil
}

func (s *LoopService) Run(ctx context.Context, dueHandler DueHandler) error {

	for ctx.Err() == nil {

		due, next, hasNext, err := s.schedulerStorage.ClaimDueTasks(ctx, defaultClaimLimit)
		if err != nil {
			s.logger.Error("failed to claim due tasks", logger.Error(err))
			sleepCtx(ctx, defaultSleepDuration)
			continue
		}

		err = s.runIteration(ctx, due, dueHandler)
		if err != nil {
			s.logger.Error("failed to run iteration", logger.Error(err))
			sleepCtx(ctx, defaultSleepDuration)
			continue
		}

		sleepCtx(ctx, getSleepDuration(next, hasNext))
	}

	return ctx.Err()
}
