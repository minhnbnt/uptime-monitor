package events

import (
	"context"
	"log/slog"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis/consumer"
)

const ackFlushInterval = 1 * time.Second

type LazyAckBatcher struct {
	acker     *consumer.StreamEventConsumer
	ch        chan *dto.DebeziumMessage
	batchSize int
	logger    *slog.Logger
}

func NewLazyAckBatcher(acker *consumer.StreamEventConsumer, batchSize int, logger *slog.Logger) *LazyAckBatcher {

	if batchSize < 1 {
		batchSize = 10
	}

	return &LazyAckBatcher{
		acker:     acker,
		ch:        make(chan *dto.DebeziumMessage, batchSize),
		batchSize: batchSize,
		logger:    logger,
	}
}

func RegisterLazyAckBatcher(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*LazyAckBatcher, error) {

		cfg := do.MustInvoke[*config.Config](i)
		acker := do.MustInvoke[*consumer.StreamEventConsumer](i)
		logger := do.MustInvoke[*slog.Logger](i)

		return NewLazyAckBatcher(
			acker,
			cfg.Redis.StreamAckBatchSize,
			logger,
		), nil
	})
}

func (l *LazyAckBatcher) Ack(ctx context.Context, event *dto.DebeziumMessage) error {

	select {
	case l.ch <- event:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *LazyAckBatcher) flush(ctx context.Context, items []*dto.DebeziumMessage) {

	if err := l.acker.AckBatch(ctx, items); err != nil {
		l.logger.Error("lazy ack flush failed", slog.Int("count", len(items)), slog.Any("error", err))
	}
}

func (l *LazyAckBatcher) Run(ctx context.Context) {

	for ctx.Err() == nil {

		batchCtx, cancel := context.WithTimeout(ctx, ackFlushInterval)
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
