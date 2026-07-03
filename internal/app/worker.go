package app

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	digesthandler "github.com/minhnbnt/uptime-monitor/internal/features/digest/handler"
	pinghandler "github.com/minhnbnt/uptime-monitor/internal/features/ping/handler"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

func RunPingWorker(ctx context.Context, i do.Injector) {

	cfg := do.MustInvoke[*config.Config](i)
	log := do.MustInvoke[logger.Logger](i)

	switch cfg.Scheduler.Backend {
	case "redis":
		runRedisPingWorker(ctx, i)

	case "temporal":
		runTemporalWorker(ctx, i)

	default:
		log.Panic(
			"unknown scheduler backend",
			logger.String("backend", cfg.Scheduler.Backend),
		)
	}
}

func RunDigestWorker(ctx context.Context, i do.Injector) {

	digest := do.MustInvoke[*digesthandler.DigestWorkerRunner](i)
	log := do.MustInvoke[logger.Logger](i)

	err := digest.RunDigestWorker(ctx)
	if err != nil {
		log.Panic("Digest worker failed", logger.Error(err))
	}
}

func runTemporalWorker(ctx context.Context, i do.Injector) {

	temporal := do.MustInvoke[*pinghandler.TemporalWorkerRunner](i)
	log := do.MustInvoke[logger.Logger](i)

	err := temporal.RunTemporalWorker(ctx)
	if err != nil {
		log.Panic("Temporal worker failed", logger.Error(err))
	}
}

func runRedisPingWorker(ctx context.Context, i do.Injector) {
	runner := do.MustInvoke[*pinghandler.ZSetWorkerRunner](i)
	runner.RunZSetWorker(ctx)
}
