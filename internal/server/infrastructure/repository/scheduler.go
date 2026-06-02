package repository

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/samber/do/v2"
	temporalclient "go.temporal.io/sdk/client"

	temporalcfg "github.com/minhnbnt/uptime-monitor/internal/config/temporal"
	"github.com/minhnbnt/uptime-monitor/internal/server/domain"
)

type PingSchedulerRepository struct {
	client temporalclient.ScheduleClient

	taskQueue string
	workflow  string
}

func RegisterPingSchedulerRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingSchedulerRepository, error) {

		clientWrapper := do.MustInvoke[*temporalcfg.ClientWrapper](i)
		temporalCfg := do.MustInvoke[*temporalcfg.Config](i)
		schedulerClient := clientWrapper.GetClient().ScheduleClient()

		return &PingSchedulerRepository{
			client:    schedulerClient,
			taskQueue: temporalCfg.TaskQueue,
			workflow:  temporalCfg.Workflow,
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

func (psr *PingSchedulerRepository) NewScheduler(ctx context.Context, endpoint *domain.Endpoint) error {

	id := fmt.Sprintf("%d", endpoint.ID)
	offset := calculateOffset(id, endpoint.Interval)

	scheduleOptions := temporalclient.ScheduleOptions{

		ID: toScheduleID(id),

		Spec: temporalclient.ScheduleSpec{
			Intervals: []temporalclient.ScheduleIntervalSpec{
				{Every: endpoint.Interval, Offset: offset},
			},
		},

		Action: &temporalclient.ScheduleWorkflowAction{
			TaskQueue: psr.taskQueue,
			Workflow:  psr.workflow,
			Args: []any{
				endpoint.ID,
				endpoint.Method,
				endpoint.URL,
				endpoint.ExpectedCode,
			},
		},
	}

	_, err := psr.client.Create(ctx, scheduleOptions)
	return err
}
