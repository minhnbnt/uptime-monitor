package handler

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"
	temporalworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	pingservice "github.com/minhnbnt/uptime-monitor/internal/features/ping/service"
	pingworkflow "github.com/minhnbnt/uptime-monitor/internal/features/ping/workflow"
)

type TemporalWorkerRunner struct {
	worker      temporalworker.Worker
	taskQueue   string
	pingService *pingservice.PingService
	logger      *slog.Logger
}

func RegisterTemporalWorkerRunner(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TemporalWorkerRunner, error) {

		clientWrapper := do.MustInvoke[*config.TemporalClientWrapper](i)
		cfg := do.MustInvoke[*config.Config](i)
		pingService := do.MustInvoke[*pingservice.PingService](i)
		logger := do.MustInvoke[*slog.Logger](i)

		taskQueue := cfg.Temporal.TaskQueue

		client := clientWrapper.GetClient()
		worker := temporalworker.New(client, taskQueue, temporalworker.Options{})

		return &TemporalWorkerRunner{
			worker:      worker,
			taskQueue:   taskQueue,
			pingService: pingService,
			logger:      logger,
		}, nil
	})
}

func (wr *TemporalWorkerRunner) RunTemporalWorker(ctx context.Context) error {

	worker := wr.worker

	worker.RegisterWorkflowWithOptions(
		pingworkflow.PingWorkflow,
		workflow.RegisterOptions{Name: "ping-workflow"},
	)
	worker.RegisterActivity(wr.pingService.Ping)
	worker.RegisterActivity(wr.pingService.Record)

	shutdownChan := make(chan any)
	go func() {
		defer close(shutdownChan)
		<-ctx.Done()
	}()

	return worker.Run(shutdownChan)
}
