package handler

import (
	"context"
	"log/slog"
	"sync"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/service"
)

type PingService interface {
	Run(ctx context.Context, channel <-chan dto.PingTask)
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
			loopService: do.MustInvoke[*service.ZsetLoopService](i),
			pingService: do.MustInvoke[*service.PingLoopService](i),
			logger:      do.MustInvoke[*slog.Logger](i),
			config:      do.MustInvoke[*config.Config](i),
		}, nil
	})
}

func (r *ZSetWorkerRunner) RunZSetWorker(ctx context.Context) {

	channel := make(chan dto.PingTask, 20)
	defer close(channel)

	handler := func(_ context.Context, tasks []dto.PingTask) {
		for _, t := range tasks {
			channel <- t
		}
	}

	claimLimit := int64(r.config.Redis.SchedulerClaimLimit)
	if claimLimit < 1 {
		claimLimit = 10
	}

	waitgroup := sync.WaitGroup{}
	defer waitgroup.Wait()

	workers := r.config.Redis.SchedulerWorkers
	if workers < 1 {
		workers = 10
	}

	for range workers {
		waitgroup.Go(func() { r.pingService.Run(ctx, channel) })
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
