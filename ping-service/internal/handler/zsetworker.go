package handler

import (
	"context"
	"iter"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/service"
)

type PingService interface {
	Ping(ctx context.Context, ep *domain.Endpoint) (bool, error)
	Record(ctx context.Context, event *domain.ServerEvent) error
}

type LoopRunner interface {
	Run(ctx context.Context, shardID uint, claimLimit int64, dueHandler service.DueHandler)
}

type ZSetWorkerRunner struct {
	loopService LoopRunner
	pingService PingService
	logger      *slog.Logger
	config      *config.Config
}

func RegisterZSetWorkerRunner(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ZSetWorkerRunner, error) {
		return &ZSetWorkerRunner{
			loopService: do.MustInvoke[*service.LoopService](i),
			pingService: do.MustInvoke[*service.PingService](i),
			logger:      do.MustInvoke[*slog.Logger](i),
			config:      do.MustInvoke[*config.Config](i),
		}, nil
	})
}

func (r *ZSetWorkerRunner) RunZSetWorker(ctx context.Context) {

	channel := make(chan *domain.Endpoint, 20)
	defer close(channel)

	handler := func(ctx context.Context, endpoints iter.Seq[*domain.Endpoint]) {
		for ep := range endpoints {
			channel <- ep
		}
	}

	claimLimit := int64(r.config.Redis.SchedulerClaimLimit)
	if claimLimit < 1 {
		claimLimit = 10
	}

	waitgroup := sync.WaitGroup{}
	defer waitgroup.Done()

	for range 10 {
		waitgroup.Go(func() { r.runWorkerLoop(ctx, channel) })
	}

	shardCount := max(r.config.Redis.SchedulerShards, 1)
	for shardID := range shardCount {
		waitgroup.Go(func() {
			r.loopService.Run(
				ctx, uint(shardID),
				claimLimit, handler,
			)
		})
	}
}

func (r *ZSetWorkerRunner) runWorkerLoop(ctx context.Context, channel <-chan *domain.Endpoint) {

	for {
		select {
		case ep, ok := <-channel:
			if !ok {
				return
			}

			r.pingAndRecordEndpoint(ctx, ep)

		case <-ctx.Done():
			return
		}
	}
}

func (r *ZSetWorkerRunner) pingAndRecordEndpoint(ctx context.Context, ep *domain.Endpoint) {

	isUp, pingErr := r.pingService.Ping(ctx, ep)

	if pingErr != nil {
		r.logger.Warn(
			"ping failed",
			slog.String("url", ep.URL),
			slog.String("method", ep.Method),
			slog.Int("expected_code", ep.ExpectedCode),
			slog.Int64("endpoint_id", int64(ep.ID)),
			slog.Any("error", pingErr),
		)
	}

	status := domain.StatusOn
	if pingErr != nil || !isUp {
		status = domain.StatusOff
	}

	id, err := uuid.NewV7()
	if err != nil {
		r.logger.Error(
			"generate event id",
			slog.Int64("endpoint", int64(ep.ID)),
			slog.Any("error", err),
		)
		return
	}

	event := domain.ServerEvent{
		ID:         id,
		EndpointID: ep.ID,
		Status:     status,
	}

	if err := r.pingService.Record(ctx, &event); err != nil {
		r.logger.Error(
			"record event",
			slog.Int64("endpoint", int64(ep.ID)),
			slog.Any("error", err),
		)
	}
}
