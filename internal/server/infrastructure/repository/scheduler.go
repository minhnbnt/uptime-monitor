package repository

import (
	"context"
	"hash/fnv"
	"os"
	"time"

	"github.com/samber/do/v2"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/minhnbnt/uptime-monitor/internal/config"
)

type PingSchedulerRepository struct {
	client temporalclient.ScheduleClient

	taskQueue string
	workflow  string
}

func RegisterPingSchedulerRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingSchedulerRepository, error) {

		clientWrapper := do.MustInvoke[*config.TemporalClientWrapper](i)
		schedulerClient := clientWrapper.GetClient().ScheduleClient()

		return &PingSchedulerRepository{
			client:    schedulerClient,
			taskQueue: os.Getenv("TEMPORAL_TASK_QUEUE"),
			workflow:  os.Getenv("TEMPORAL_WORKFLOW_NAME"),
		}, nil
	})
}

func toScheduleID(serverID string) string {
	return "ping-schedule-" + serverID
}

func calculateOffset(id string, interval time.Duration) time.Duration {

	hasher := fnv.New64a()
	hasher.Write([]byte(id))

	offset := hasher.Sum64() % uint64(interval)
	return time.Duration(offset)
}

func (psr *PingSchedulerRepository) NewScheduler(ctx context.Context, id string, interval time.Duration) error {

	offset := calculateOffset(id, interval)

	scheduleOptions := temporalclient.ScheduleOptions{

		ID: toScheduleID(id),

		Spec: temporalclient.ScheduleSpec{
			Intervals: []temporalclient.ScheduleIntervalSpec{
				{Every: interval, Offset: offset},
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
