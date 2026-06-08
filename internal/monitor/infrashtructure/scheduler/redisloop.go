package scheduler

import (
	"context"
	"iter"
	"maps"
	"time"

	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

type RedisWorkerLoop struct {
	logger logger.Logger

	schedulerStorage *ZSetScheduleRepository
	scoreUpdater     *ScoreUpdater
	endpointProvider *EndpointProvider
}

const defaultClaimLimit = 50
const defaultSleepDuration = 5 * time.Second

func sleepCtx(ctx context.Context, d time.Duration) {

	if d <= 0 {
		return
	}

	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func getSleepDuration(next *ScheduledTask) time.Duration {

	if next == nil {
		return defaultSleepDuration
	}

	nextTime := time.UnixMilli(next.Score)
	if nextTime.Before(time.Now()) {
		return 0
	}

	return time.Until(nextTime)
}

func (r *RedisWorkerLoop) runIteration(ctx context.Context, due []ScheduledTask, dueHandler DueHandler) error {

	ids := lo.Map(due, func(task ScheduledTask, _ int) uint { return task.EndpointID })
	endpointMap, err := r.endpointProvider.GetBatch(ctx, ids)
	if err != nil {
		return err
	}

	endpoints := maps.Values(endpointMap)
	dueHandler(ctx, endpoints)

	updates := lo.SliceToMap(due, func(task ScheduledTask) (uint, int64) {
		nextSchedule := task.Score + endpointMap[task.EndpointID].Interval.Milliseconds()
		return task.EndpointID, nextSchedule
	})

	if len(updates) > 0 {
		return r.scoreUpdater.UpdateBatch(ctx, updates)
	}

	return nil
}

type DueHandler func(ctx context.Context, tasks iter.Seq[*domain.Endpoint])

func (r *RedisWorkerLoop) Run(ctx context.Context, dueHandler DueHandler) error {

	for ctx.Err() == nil {

		due, next, err := r.schedulerStorage.ClaimDueTasks(ctx, defaultClaimLimit)
		if err != nil {
			r.logger.Error("failed to claim due tasks", logger.Error(err))
			sleepCtx(ctx, defaultSleepDuration)
			continue
		}

		err = r.runIteration(ctx, due, dueHandler)
		if err != nil {
			r.logger.Error("failed to run iteration", logger.Error(err))
			sleepCtx(ctx, defaultSleepDuration)
			continue
		}

		sleepCtx(ctx, getSleepDuration(next))
	}

	return ctx.Err()
}
