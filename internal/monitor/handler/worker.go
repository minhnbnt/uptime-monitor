package handler

import (
	"context"

	temporalworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	infra "github.com/minhnbnt/uptime-monitor/internal/monitor/infrashtructure"
)

type TemporalWorkerRunner struct {
	worker     temporalworker.Worker
	pingWorker infra.PingWorker
}

func (wr *TemporalWorkerRunner) RunPingWorker() {

	worker := wr.worker

	worker.RegisterWorkflow(func(ctx workflow.Context, id uint) error {

		result := ""
		if err := workflow.ExecuteActivity(ctx, fetchEndpointInfo, id).Get(ctx, &result); err != nil {
			return err
		}

		return nil
	})

	worker.RegisterActivity(fetchEndpointInfo)
}

func fetchEndpointInfo(ctx context.Context, id uint) (string, error) {
	return "https://echo.hoppscotch.io", nil
}
