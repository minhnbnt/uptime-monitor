package scheduler

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	temporalclient "go.temporal.io/sdk/client"

	temporalcfg "github.com/minhnbnt/uptime-monitor/internal/config/temporal"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

type SchedulerRepository interface {
	Register(ctx context.Context, endpoint *domain.Endpoint) error
	Unregister(ctx context.Context, endpointID uint) error
}

type TemporalSchedulerRepository struct {
	client temporalclient.ScheduleClient

	taskQueue string
	workflow  string
}

func RegisterTemporalSchedulerRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TemporalSchedulerRepository, error) {

		clientWrapper := do.MustInvoke[*temporalcfg.ClientWrapper](i)
		temporalCfg := do.MustInvoke[*temporalcfg.Config](i)
		schedulerClient := clientWrapper.GetClient().ScheduleClient()

		return &TemporalSchedulerRepository{
			client:    schedulerClient,
			taskQueue: temporalCfg.TaskQueue,
			workflow:  temporalCfg.WorkflowName,
		}, nil
	})
}

func toScheduleID(serverID string) string {
	return "ping-schedule-" + serverID
}

func (tsr *TemporalSchedulerRepository) Register(ctx context.Context, endpoint *domain.Endpoint) error {

	id := fmt.Sprint(endpoint.ID)
	offset := utils.GenerateOffset(id, endpoint.Interval)

	scheduleOptions := temporalclient.ScheduleOptions{

		ID: toScheduleID(id),

		Spec: temporalclient.ScheduleSpec{
			Intervals: []temporalclient.ScheduleIntervalSpec{
				{Every: endpoint.Interval, Offset: offset},
			},
		},

		Action: &temporalclient.ScheduleWorkflowAction{
			TaskQueue: tsr.taskQueue,
			Workflow:  tsr.workflow,
			Args: []any{
				endpoint.ID,
				endpoint.Method,
				endpoint.URL,
				endpoint.ExpectedCode,
			},
		},
	}

	_, err := tsr.client.Create(ctx, scheduleOptions)
	return err
}

func (tsr *TemporalSchedulerRepository) Unregister(ctx context.Context, endpointID uint) error {

	scheduleID := toScheduleID(fmt.Sprint(endpointID))

	handle := tsr.client.GetHandle(ctx, scheduleID)
	return handle.Delete(ctx)
}
