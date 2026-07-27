package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/samber/do/v2"
	"go.temporal.io/api/serviceerror"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
)

type TemporalDigestStarter struct {
	scheduleClient temporalclient.ScheduleClient
	client         temporalclient.Client
	taskQueue      string
	workflowName   string
}

func RegisterDigestStarter(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TemporalDigestStarter, error) {

		clientWrapper := do.MustInvoke[*config.TemporalClientWrapper](i)
		cfg := do.MustInvoke[*config.Config](i)

		client := clientWrapper.GetClient()
		scheduleClient := client.ScheduleClient()

		return &TemporalDigestStarter{
			client:         client,
			scheduleClient: scheduleClient,
			taskQueue:      cfg.Temporal.DigestTaskQueue,
			workflowName:   cfg.Temporal.WorkflowName,
		}, nil
	})
}

func getScheduleID(id uint) string {
	return fmt.Sprintf("digest-user-%d", id)
}

func (ds *TemporalDigestStarter) StartDigest(ctx context.Context, userID uint) error {

	from := time.Now().Add(-30 * 24 * time.Hour)

	_, err := ds.client.ExecuteWorkflow(
		ctx,
		temporalclient.StartWorkflowOptions{TaskQueue: ds.taskQueue},
		ds.workflowName,
		userID,
		from,
	)

	return err
}

func (ds *TemporalDigestStarter) DescribeSchedule(ctx context.Context, userID uint) (*domain.ScheduleInfo, error) {

	scheduleID := getScheduleID(userID)
	handle := ds.scheduleClient.GetHandle(ctx, scheduleID)

	desc, err := handle.Describe(ctx)
	if _, ok := errors.AsType[*serviceerror.NotFound](err); ok {
		return &domain.ScheduleInfo{}, nil
	}

	if err != nil {
		return nil, err
	}

	spec := desc.Schedule.Spec
	info := &domain.ScheduleInfo{
		Exists:   true,
		FromDate: spec.StartAt,
		ToDate:   spec.EndAt,
	}

	if len(spec.Calendars) > 0 {

		cal := spec.Calendars[0]
		hour, minute := 8, 0

		if len(cal.Hour) > 0 {
			hour = cal.Hour[0].Start
		}

		if len(cal.Minute) > 0 {
			minute = cal.Minute[0].Start
		}

		info.DigestTime = fmt.Sprintf("%02d:%02d", hour, minute)
	}

	return info, nil
}

func (ds *TemporalDigestStarter) UpsertSchedule(ctx context.Context, userID uint, cfg domain.ScheduleConfig) error {

	scheduleID := getScheduleID(userID)

	hour, err := strconv.Atoi(cfg.DigestTime[:2])
	if err != nil {
		return err
	}

	minute, err := strconv.Atoi(cfg.DigestTime[3:])
	if err != nil {
		return err
	}

	spec := temporalclient.ScheduleSpec{
		StartAt: cfg.FromDate, EndAt: cfg.ToDate,
		Calendars: []temporalclient.ScheduleCalendarSpec{{
			Hour:   []temporalclient.ScheduleRange{{Start: hour}},
			Minute: []temporalclient.ScheduleRange{{Start: minute}},
		}},
	}

	action := &temporalclient.ScheduleWorkflowAction{
		Workflow:  ds.workflowName,
		TaskQueue: ds.taskQueue,
		Args:      []any{userID, cfg.FromDate},
	}

	handle := ds.scheduleClient.GetHandle(ctx, scheduleID)
	if _, err := handle.Describe(ctx); err != nil {
		_, err = ds.scheduleClient.Create(ctx, temporalclient.ScheduleOptions{
			ID:     scheduleID,
			Spec:   spec,
			Action: action,
		})
		return err
	}

	return handle.Update(ctx, temporalclient.ScheduleUpdateOptions{
		DoUpdate: func(_ temporalclient.ScheduleUpdateInput) (*temporalclient.ScheduleUpdate, error) {
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

func (ds *TemporalDigestStarter) DeleteSchedule(ctx context.Context, userID uint) error {

	scheduleID := getScheduleID(userID)
	handle := ds.scheduleClient.GetHandle(ctx, scheduleID)

	return handle.Delete(ctx)
}
