package handler

import (
	"context"

	"github.com/samber/do/v2"
	temporalworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	temporalcfg "github.com/minhnbnt/uptime-monitor/internal/config/temporal"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/monitor/services"
)

type TemporalWorkerRunner struct {
	worker      temporalworker.Worker
	pingService *services.PingService

	logger logger.Logger
}

func RegisterTemporalWorkerRunner(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TemporalWorkerRunner, error) {

		clientWrapper := do.MustInvoke[*temporalcfg.ClientWrapper](i)
		temporalCfg := do.MustInvoke[*temporalcfg.Config](i)
		pingService := do.MustInvoke[*services.PingService](i)
		logger := do.MustInvoke[logger.Logger](i)

		client := clientWrapper.GetClient()
		worker := temporalworker.New(client, temporalCfg.TaskQueue, temporalworker.Options{})

		return &TemporalWorkerRunner{
			worker:      worker,
			pingService: pingService,
			logger:      logger,
		}, nil
	})
}

func (wr *TemporalWorkerRunner) RunTemporalWorker(ctx context.Context) {

	worker := wr.worker

	worker.RegisterWorkflowWithOptions(
		wr.pingService.PingWorkflow,
		workflow.RegisterOptions{Name: "ping-workflow"},
	)

	worker.RegisterActivity(wr.pingService.Ping)
	worker.RegisterActivity(wr.pingService.Record)

	shutdownChan := make(chan any)

	go func() {
		defer close(shutdownChan)
		<-ctx.Done()
	}()

	if err := worker.Run(shutdownChan); err != nil {
		wr.logger.Error("Temporal worker failed", logger.Error(err))
	}
}
