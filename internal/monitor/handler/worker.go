package handler

import (
	"context"
	"time"

	"github.com/samber/do/v2"
	temporalworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	temporalcfg "github.com/minhnbnt/uptime-monitor/internal/config/temporal"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	infra "github.com/minhnbnt/uptime-monitor/internal/monitor/infrashtructure"
)

type TemporalWorkerRunner struct {
	worker     temporalworker.Worker
	pingWorker *infra.PingWorker

	logger logger.Logger
}

func RegisterTemporalWorkerRunner(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TemporalWorkerRunner, error) {

		clientWrapper := do.MustInvoke[*temporalcfg.ClientWrapper](i)
		temporalCfg := do.MustInvoke[*temporalcfg.Config](i)
		pingWorker := do.MustInvoke[*infra.PingWorker](i)
		logger := do.MustInvoke[logger.Logger](i)

		client := clientWrapper.GetClient()
		worker := temporalworker.New(client, temporalCfg.TaskQueue, temporalworker.Options{})

		return &TemporalWorkerRunner{
			worker:     worker,
			pingWorker: pingWorker,
			logger:     logger,
		}, nil
	})
}

func (wr *TemporalWorkerRunner) PingWorkflow(ctx workflow.Context, method string, url string, expectedCode int) error {

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	statusCode := 0
	if err := workflow.ExecuteActivity(ctx, wr.pingWorker.Ping, method, url).Get(ctx, &statusCode); err != nil {
		wr.logger.Warn(
			"failed to ping server",
			logger.String("method", method),
			logger.String("url", url),
			logger.Error(err),
		)
		return nil
	}

	if statusCode != expectedCode {
		wr.logger.Warn(
			"unexpected status code",
			logger.Int("expected", expectedCode),
			logger.Int("got", statusCode),
		)
		return nil
	}

	return nil
}

func (wr *TemporalWorkerRunner) RunTemporalWorker(ctx context.Context) {

	worker := wr.worker

	worker.RegisterWorkflowWithOptions(
		wr.PingWorkflow,
		workflow.RegisterOptions{Name: "ping-workflow"},
	)

	worker.RegisterActivity(wr.pingWorker.Ping)

	shutdownChan := make(chan any)

	go func() {
		defer close(shutdownChan)
		<-ctx.Done()
	}()

	worker.Run(shutdownChan)
}
