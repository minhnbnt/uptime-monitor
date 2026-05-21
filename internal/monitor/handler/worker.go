package handler

import (
	"os"
	"time"

	"github.com/samber/do/v2"
	temporalworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	infra "github.com/minhnbnt/uptime-monitor/internal/monitor/infrashtructure"
	"github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/logger"
)

type TemporalWorkerRunner struct {
	worker       temporalworker.Worker
	pingWorker   *infra.PingWorker
	shutdownChan chan any

	logger logger.Logger
}

func RegisterTemporalWorkerRunner(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TemporalWorkerRunner, error) {

		clientWrapper := do.MustInvoke[*config.TemporalClientWrapper](i)
		pingWorker := do.MustInvoke[*infra.PingWorker](i)
		logger := do.MustInvoke[logger.Logger](i)

		client := clientWrapper.GetClient()
		taskQueue := os.Getenv("TEMPORAL_TASK_QUEUE")
		worker := temporalworker.New(client, taskQueue, temporalworker.Options{})

		channel := make(chan any, 1)

		return &TemporalWorkerRunner{
			worker:       worker,
			pingWorker:   pingWorker,
			shutdownChan: channel,
			logger:       logger,
		}, nil
	})
}

func (wr *TemporalWorkerRunner) Shutdown() {
	wr.shutdownChan <- nil
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

func (wr *TemporalWorkerRunner) RunTemporalWorker() {

	worker := wr.worker

	worker.RegisterWorkflowWithOptions(
		wr.PingWorkflow,
		workflow.RegisterOptions{Name: "ping-workflow"},
	)

	worker.RegisterActivity(wr.pingWorker.Ping)

	worker.Run(wr.shutdownChan)
}
