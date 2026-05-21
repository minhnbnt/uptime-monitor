package handler

import (
	"fmt"

	temporalworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	infra "github.com/minhnbnt/uptime-monitor/internal/monitor/infrashtructure"
)

type TemporalWorkerRunner struct {
	worker     temporalworker.Worker
	pingWorker infra.PingWorker
}

// TODO: add Register function by do

func (wr *TemporalWorkerRunner) RunTemporalWorker() {

	worker := wr.worker

	worker.RegisterWorkflow(func(ctx workflow.Context, method string, url string, expectedCode int) error {

		statusCode := 0
		if err := workflow.ExecuteActivity(ctx, wr.pingWorker.Ping, method, url).Get(ctx, &statusCode); err != nil {
			return err
		}

		if statusCode != expectedCode {
			return fmt.Errorf("unexpected status code, expected: %d, got: %d", expectedCode, statusCode)
		}

		return nil
	})

	worker.RegisterActivity(wr.pingWorker.Ping)
}
