package handler

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"
	temporalworker "go.temporal.io/sdk/worker"
	temporalworkflow "go.temporal.io/sdk/workflow"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/service"
	digestworkflow "github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure/workflow"
)

type DigestWorkerRunner struct {
	worker        temporalworker.Worker
	digestService *service.DigestService
	logger        *slog.Logger
}

func RegisterDigestWorkerRunner(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*DigestWorkerRunner, error) {

		clientWrapper := do.MustInvoke[*config.TemporalClientWrapper](i)
		digestService := do.MustInvoke[*service.DigestService](i)

		logger := do.MustInvoke[*slog.Logger](i)
		cfg := do.MustInvoke[*config.Config](i)

		client := clientWrapper.GetClient()
		worker := temporalworker.New(
			client,
			cfg.Temporal.DigestTaskQueue,
			temporalworker.Options{},
		)

		return &DigestWorkerRunner{
			worker:        worker,
			digestService: digestService,
			logger:        logger,
		}, nil
	})
}

func (wr *DigestWorkerRunner) RunDigestWorker(ctx context.Context) error {

	worker := wr.worker

	worker.RegisterWorkflowWithOptions(
		digestworkflow.SendReportWorkflow,
		temporalworkflow.RegisterOptions{Name: "send-report"},
	)

	worker.RegisterActivity(wr.digestService.SendUserDigest)

	shutdownChan := make(chan any)
	go func() {
		defer close(shutdownChan)
		<-ctx.Done()
	}()

	return worker.Run(shutdownChan)
}
