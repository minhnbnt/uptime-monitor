package handler

import (
	"context"
	"log/slog"
	"sync"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/service"
)

type StaleLoopRunner interface {
	Run(ctx context.Context, shardID uint, claimLimit int64)
}

type StaleWorkerRunner struct {
	loopService StaleLoopRunner
	logger      *slog.Logger
	config      *config.Config
}

func RegisterStaleWorkerRunner(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*StaleWorkerRunner, error) {
		return &StaleWorkerRunner{
			loopService: do.MustInvoke[*service.StaleLoopService](i),
			logger:      do.MustInvoke[*slog.Logger](i),
			config:      do.MustInvoke[*config.Config](i),
		}, nil
	})
}

func (r *StaleWorkerRunner) RunStaleWorker(ctx context.Context) {

	claimLimit := int64(r.config.Redis.SchedulerClaimLimit)
	if claimLimit < 1 {
		claimLimit = 10
	}

	waitgroup := sync.WaitGroup{}
	defer waitgroup.Wait()

	shardCount := max(r.config.Redis.SchedulerShards, 1)
	for shardID := range shardCount {
		waitgroup.Go(func() {
			r.loopService.Run(ctx, uint(shardID), claimLimit)
		})
	}
}
