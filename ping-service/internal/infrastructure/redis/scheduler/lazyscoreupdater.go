package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
)

const flushInterval = 1 * time.Second

type scoreUpdate struct {
	serverID  uint
	nextScore int64
}

type LazyScoreUpdater struct {
	updater   *ScoreUpdater
	ch        chan scoreUpdate
	batchSize int
	logger    *slog.Logger
}

func NewLazyScoreUpdater(updater *ScoreUpdater, batchSize int, logger *slog.Logger) *LazyScoreUpdater {

	if batchSize < 1 {
		batchSize = 50
	}

	return &LazyScoreUpdater{
		updater:   updater,
		ch:        make(chan scoreUpdate, batchSize),
		batchSize: batchSize,
		logger:    logger,
	}
}

func RegisterLazyScoreUpdater(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*LazyScoreUpdater, error) {

		cfg := do.MustInvoke[*config.Config](i)
		updater := do.MustInvoke[*ScoreUpdater](i)
		logger := do.MustInvoke[*slog.Logger](i)

		return NewLazyScoreUpdater(
			updater,
			cfg.Redis.SchedulerUpdateBatchSize,
			logger,
		), nil
	})
}

func (l *LazyScoreUpdater) Update(ctx context.Context, serverID uint, nextScore int64) error {

	select {
	case l.ch <- scoreUpdate{serverID: serverID, nextScore: nextScore}:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *LazyScoreUpdater) flush(ctx context.Context, items []scoreUpdate) {

	scoreMap := lo.SliceToMap(items, func(it scoreUpdate) (uint, int64) {
		return it.serverID, it.nextScore
	})

	if err := l.updater.UpdateBatch(ctx, scoreMap); err != nil {
		l.logger.Error("lazy score flush failed", slog.Int("count", len(items)), slog.Any("error", err))
	}
}

func (l *LazyScoreUpdater) Run(ctx context.Context) {
	for ctx.Err() == nil {

		batchCtx, cancel := context.WithTimeout(ctx, flushInterval)
		items, _, _, ok := lo.BufferWithContext(batchCtx, l.ch, l.batchSize)
		cancel()

		if len(items) > 0 {
			l.flush(ctx, items)
		}

		if !ok {
			return
		}
	}
}
