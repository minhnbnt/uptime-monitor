package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	pinginfra "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
	pingrepo "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/repository"
	scheduler "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
)

// staleMaxSleep caps one idle wait of the stale loop.
const staleMaxSleep = 30 * time.Second

type staleStore interface {
	Remove(ctx context.Context, endpointID uint) error
	ClaimOverdue(ctx context.Context, shardID uint, limit int64) (
		due []scheduler.ScheduledTask,
		next scheduler.ScheduledTask,
		hasNext bool,
		err error,
	)
}

type StaleLoopService struct {
	logger    *slog.Logger
	freshness staleStore
	recorder  recordWorker
}

func RegisterStaleLoopService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*StaleLoopService, error) {
		return &StaleLoopService{
			logger:    do.MustInvoke[*slog.Logger](i),
			freshness: do.MustInvoke[*pingrepo.FreshnessStore](i),
			recorder:  do.MustInvoke[*pinginfra.RecordStatusWorker](i),
		}, nil
	})
}

func staleSleepDuration(next scheduler.ScheduledTask, hasNext bool) time.Duration {

	if !hasNext {
		return staleMaxSleep
	}

	d := time.Until(time.UnixMilli(next.Score))
	if d <= 0 {
		return 0
	}

	return min(d, staleMaxSleep)
}

func (s *StaleLoopService) markUnknown(ctx context.Context, task scheduler.ScheduledTask) {

	id, err := uuid.NewV7()
	if err != nil {
		s.logger.Error(
			"generate event id",
			slog.Int64("endpoint", int64(task.EndpointID)),
			slog.Any("error", err),
		)
		return
	}

	event := domain.ServerEvent{
		ID:         id,
		EndpointID: task.EndpointID,
		Status:     domain.StatusUnknown,
	}

	// On failure the entry stays claimed at now+10s and is retried later;
	// on success it leaves the set until the agent pushes again.
	if err := s.recorder.Record(ctx, &event, PushStaleInterval); err != nil {
		s.logger.Warn(
			"failed to record stale event",
			slog.Int64("endpoint", int64(task.EndpointID)),
			slog.Any("error", err),
		)
		return
	}

	if err := s.freshness.Remove(ctx, task.EndpointID); err != nil {
		s.logger.Error(
			"failed to remove stale entry",
			slog.Int64("endpoint", int64(task.EndpointID)),
			slog.Any("error", err),
		)
	}
}

func (s *StaleLoopService) Run(ctx context.Context, shardID uint, claimLimit int64) {

	for ctx.Err() == nil {

		due, next, hasNext, err := s.freshness.ClaimOverdue(ctx, shardID, claimLimit)
		if err != nil {
			s.logger.Error("failed to claim overdue entries", slog.Any("error", err))
			sleepCtx(ctx, defaultSleepDuration)
			continue
		}

		for _, task := range due {
			s.markUnknown(ctx, task)
		}

		if len(due) != int(claimLimit) {
			sleepCtx(ctx, staleSleepDuration(next, hasNext))
		}
	}
}
