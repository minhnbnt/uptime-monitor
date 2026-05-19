package repository

import (
	"context"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/samber/do/v2"
	temporalclient "go.temporal.io/sdk/client"
)

type PingSchedulerRepository struct {
	client temporalclient.ScheduleClient

	taskQueue string
	workflow  string
}

func RegisterPingSchedulerRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingSchedulerRepository, error) {

		clientWrapper := do.MustInvoke[config.TemporalClientWrapper](i)
		schedulerClient := clientWrapper.GetClient().ScheduleClient()

		return &PingSchedulerRepository{
			client: schedulerClient,
			// TODO: fill others fields
		}, nil
	})
}

func toSchredulerID(serverID string) string {
	return "ping-scheduler" + serverID
}

func (psr *PingSchedulerRepository) NewScheduler(ctx context.Context, id string, duration time.Duration) error {

	scheduleOptions := temporalclient.ScheduleOptions{

		ID: toSchredulerID(id),

		Spec: temporalclient.ScheduleSpec{
			Intervals: []temporalclient.ScheduleIntervalSpec{
				{Every: duration},
			},
		},

		Action: &temporalclient.ScheduleWorkflowAction{
			TaskQueue: psr.taskQueue,
			Workflow:  psr.workflow,
			Args:      []any{id},
		},
	}

	_, err := psr.client.Create(ctx, scheduleOptions)
	return err
}
