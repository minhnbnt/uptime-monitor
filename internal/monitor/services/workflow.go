package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	infra "github.com/minhnbnt/uptime-monitor/internal/monitor/infrastructure"
)

type PingService struct {
	pingWorker         *infra.PingWorker
	recordStatusWorker *infra.RecordStatusWorker
}

func RegisterPingService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingService, error) {
		return &PingService{
			pingWorker:         do.MustInvoke[*infra.PingWorker](i),
			recordStatusWorker: do.MustInvoke[*infra.RecordStatusWorker](i),
		}, nil
	})
}

func (s *PingService) PingWorkflow(ctx workflow.Context, endpointID uint, method string, url string, expectedCode int) error {

	logger := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})

	statusCode := 0
	pingErr := workflow.ExecuteActivity(ctx, s.pingWorker.Ping, method, url).Get(ctx, &statusCode)

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

	return workflow.ExecuteActivity(recordCtx, s.recordStatusWorker.Record, event).Get(recordCtx, nil)
}

func (s *PingService) Ping(ctx context.Context, method, url string) (int, error) {
	return s.pingWorker.Ping(ctx, method, url)
}

func (s *PingService) Record(ctx context.Context, event *domain.ServerEvent) error {
	return s.recordStatusWorker.Record(ctx, event)
}
