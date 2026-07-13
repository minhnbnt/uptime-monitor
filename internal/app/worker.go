package app

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	digesthandler "github.com/minhnbnt/uptime-monitor/internal/features/digest/handler"
	pinghandler "github.com/minhnbnt/uptime-monitor/internal/features/ping/handler"
)

func RunPingWorker(ctx context.Context, i do.Injector) {

	cfg := do.MustInvoke[*config.Config](i)
	log := do.MustInvoke[*slog.Logger](i)

	switch cfg.Scheduler.Backend {
	case "redis":
		runRedisPingWorker(ctx, i)

	case "temporal":
		runTemporalWorker(ctx, i)

	default:
		log.Error(
			"unknown scheduler backend",
			slog.String("backend", cfg.Scheduler.Backend),
		)
		panic("unknown scheduler backend")
	}
}

func RunDigestWorker(ctx context.Context, i do.Injector) {

	digest := do.MustInvoke[*digesthandler.DigestWorkerRunner](i)
	log := do.MustInvoke[*slog.Logger](i)

	err := digest.RunDigestWorker(ctx)
	if err != nil {
		log.Error("Digest worker failed", slog.Any("error", err))
		panic(err)
	}
}

func runTemporalWorker(ctx context.Context, i do.Injector) {

	temporal := do.MustInvoke[*pinghandler.TemporalWorkerRunner](i)
	log := do.MustInvoke[*slog.Logger](i)

	err := temporal.RunTemporalWorker(ctx)
	if err != nil {
		log.Error("Temporal worker failed", slog.Any("error", err))
		panic(err)
	}
}

func runRedisPingWorker(ctx context.Context, i do.Injector) {
	runner := do.MustInvoke[*pinghandler.ZSetWorkerRunner](i)
	runner.RunZSetWorker(ctx)
}
