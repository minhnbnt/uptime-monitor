package workflow

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

func PingWorkflow(ctx workflow.Context, endpointID uint, method string, url string, expectedCode int) error {

	logger := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})

	statusCode := 0
	pingErr := workflow.ExecuteActivity(ctx, "Ping", method, url).Get(ctx, &statusCode)

	currentStatus := domain.StatusOn
	isPingOk := pingErr == nil && statusCode == expectedCode
	if !isPingOk {
		currentStatus = domain.StatusOff
	}

	logger.Info(
		"ping result",
		"url", url,
		"statusCode", statusCode,
		"isPingOk", isPingOk,
		"pingError", pingErr,
	)

	recordCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	})

	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate uuid: %w", err)
	}

	event := &domain.ServerEvent{
		ID:         id,
		EndpointID: endpointID,
		Status:     currentStatus,
	}

	return workflow.ExecuteActivity(recordCtx, "Record", event).Get(recordCtx, nil)
}
