package handler

import (
	"context"

	"github.com/samber/do/v2"
	temporalworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	temporalcfg "github.com/minhnbnt/uptime-monitor/internal/config/temporal"
	pingservice "github.com/minhnbnt/uptime-monitor/internal/features/ping/service"
	pingworkflow "github.com/minhnbnt/uptime-monitor/internal/features/ping/workflow"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

type TemporalWorkerRunner struct {
	worker      temporalworker.Worker
	taskQueue   string
	pingService *pingservice.PingService
	logger      logger.Logger
}

func RegisterTemporalWorkerRunner(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TemporalWorkerRunner, error) {

		clientWrapper := do.MustInvoke[*temporalcfg.ClientWrapper](i)
		temporalCfg := do.MustInvoke[*temporalcfg.Config](i)
		pingService := do.MustInvoke[*pingservice.PingService](i)
		logger := do.MustInvoke[logger.Logger](i)

		client := clientWrapper.GetClient()
		worker := temporalworker.New(client, temporalCfg.TaskQueue, temporalworker.Options{})

		return &TemporalWorkerRunner{
			worker:      worker,
			taskQueue:   temporalCfg.TaskQueue,
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
