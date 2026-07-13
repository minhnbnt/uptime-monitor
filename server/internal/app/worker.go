package app

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"

	digesthandler "github.com/minhnbnt/uptime-monitor/internal/features/digest/handler"
)

func RunDigestWorker(ctx context.Context, i do.Injector) {

	digest := do.MustInvoke[*digesthandler.DigestWorkerRunner](i)
	log := do.MustInvoke[*slog.Logger](i)

	err := digest.RunDigestWorker(ctx)
	if err != nil {
		log.Error("Digest worker failed", slog.Any("error", err))
		panic(err)
	}
}
