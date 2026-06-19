package infrastructure

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/samber/do/v2"
	temporalclient "go.temporal.io/sdk/client"

	temporalcfg "github.com/minhnbnt/uptime-monitor/internal/config/temporal"
)

type DigestStarter struct {
	client    temporalclient.Client
	taskQueue string
}

func RegisterDigestStarter(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*DigestStarter, error) {
		clientWrapper := do.MustInvoke[*temporalcfg.ClientWrapper](i)
		temporalCfg := do.MustInvoke[*temporalcfg.Config](i)
		return &DigestStarter{
			client:    clientWrapper.GetClient(),
			taskQueue: temporalCfg.DigestTaskQueue,
		}, nil
	})
}

func (ds *DigestStarter) StartDigest(ctx context.Context, userID uint) error {

	_, err := ds.client.ExecuteWorkflow(
		ctx,
		temporalclient.StartWorkflowOptions{TaskQueue: ds.taskQueue},
		"send-report",
		userID,
	)

	return err
}

func (ds *DigestStarter) UpsertSchedule(ctx context.Context, userID uint, fromDate, toDate time.Time, digestTime string) error {

	scheduleID := fmt.Sprintf("digest-user-%d", userID)

	hour, _ := strconv.Atoi(digestTime[:2])
	minute, _ := strconv.Atoi(digestTime[3:])

	spec := temporalclient.ScheduleSpec{
		StartAt: fromDate,
		EndAt:   toDate,
		Calendars: []temporalclient.ScheduleCalendarSpec{{
			Hour:   []temporalclient.ScheduleRange{{Start: hour}},
			Minute: []temporalclient.ScheduleRange{{Start: minute}},
		}},
	}

	action := &temporalclient.ScheduleWorkflowAction{
		Workflow:  "send-report",
		TaskQueue: ds.taskQueue,
		Args:      []any{userID},
	}

	handle := ds.client.ScheduleClient().GetHandle(ctx, scheduleID)
	_, err := handle.Describe(ctx)
	if err != nil {
		_, err = ds.client.ScheduleClient().Create(ctx, temporalclient.ScheduleOptions{
			ID:     scheduleID,
			Spec:   spec,
			Action: action,
		})
		return err
	}

	return handle.Update(ctx, temporalclient.ScheduleUpdateOptions{
		DoUpdate: func(input temporalclient.ScheduleUpdateInput) (*temporalclient.ScheduleUpdate, error) {
			return &temporalclient.ScheduleUpdate{
				Schedule: &temporalclient.Schedule{
					Spec: &spec, Action: action,
					Policy: &temporalclient.SchedulePolicies{},
					State:  &temporalclient.ScheduleState{},
				},
			}, nil
		},
	})
}

func (ds *DigestStarter) DeleteSchedule(ctx context.Context, userID uint) error {

	scheduleID := fmt.Sprintf("digest-user-%d", userID)
	handle := ds.client.ScheduleClient().GetHandle(ctx, scheduleID)

	return handle.Delete(ctx)
}
