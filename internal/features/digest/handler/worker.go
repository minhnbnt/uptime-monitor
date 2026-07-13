package handler

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"
	temporalworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	digestservice "github.com/minhnbnt/uptime-monitor/internal/features/digest/service"
	digestworkflow "github.com/minhnbnt/uptime-monitor/internal/features/digest/workflow"
)

type DigestWorkerRunner struct {
	worker        temporalworker.Worker
	digestService *digestservice.DigestService
	logger        *slog.Logger
}

func RegisterDigestWorkerRunner(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*DigestWorkerRunner, error) {

		clientWrapper := do.MustInvoke[*config.TemporalClientWrapper](i)
		config := do.MustInvoke[*config.Config](i)
		digestService := do.MustInvoke[*digestservice.DigestService](i)
		logger := do.MustInvoke[*slog.Logger](i)

		client := clientWrapper.GetClient()
		worker := temporalworker.New(
			client,
			config.Temporal.DigestTaskQueue,
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
		workflow.RegisterOptions{Name: "send-report"},
	)

	worker.RegisterActivity(wr.digestService.SendUserDigest)

	shutdownChan := make(chan any)
	go func() {
		defer close(shutdownChan)
		<-ctx.Done()
	}()

	return worker.Run(shutdownChan)
}
