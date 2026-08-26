package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

type fakeStaleStore struct {
	due       []scheduler.ScheduledTask
	next      scheduler.ScheduledTask
	hasNext   bool
	err       error
	removed   []uint
	removeErr error
}

func (f *fakeStaleStore) ClaimOverdue(
	_ context.Context, _ uint, _ int64,
) (due []scheduler.ScheduledTask, next scheduler.ScheduledTask, hasNext bool, err error) {
	return f.due, f.next, f.hasNext, f.err
}

func (f *fakeStaleStore) Remove(_ context.Context, endpointID uint) error {
	if f.removeErr != nil {
		return f.removeErr
	}
	f.removed = append(f.removed, endpointID)
	return nil
}

func TestStaleSleepDuration(t *testing.T) {
	t.Run("no next entry sleeps the max", func(t *testing.T) {
		d := staleSleepDuration(scheduler.ScheduledTask{}, false)
		if d != staleMaxSleep {
			t.Errorf("got %v, want %v", d, staleMaxSleep)
		}
	})

	t.Run("past due next entry does not sleep", func(t *testing.T) {
		past := time.Now().Add(-time.Hour).UnixMilli()
		d := staleSleepDuration(scheduler.ScheduledTask{Score: past}, true)
		if d != 0 {
			t.Errorf("got %v, want 0", d)
		}
	})

	t.Run("near future next entry sleeps until it", func(t *testing.T) {
		in5s := time.Now().Add(5 * time.Second).UnixMilli()
		d := staleSleepDuration(scheduler.ScheduledTask{Score: in5s}, true)
		if d <= 0 || d > 5*time.Second {
			t.Errorf("got %v, want ~5s", d)
		}
	})

	t.Run("far future next entry is capped at the max", func(t *testing.T) {
		in1h := time.Now().Add(time.Hour).UnixMilli()
		d := staleSleepDuration(scheduler.ScheduledTask{Score: in1h}, true)
		if d != staleMaxSleep {
			t.Errorf("got %v, want %v", d, staleMaxSleep)
		}
	})
}

func TestMarkUnknown(t *testing.T) {

	endpointID := uint(42)

	newService := func(store *fakeStaleStore, rec *mockRecordWorker) *StaleLoopService {
		return &StaleLoopService{
			logger:    logger.NewMockLogger(),
			freshness: store,
			recorder:  rec,
		}
	}

	task := scheduler.ScheduledTask{EndpointID: endpointID}

	t.Run("success records unknown and removes the entry", func(t *testing.T) {
		store := &fakeStaleStore{}
		var recorded *domain.ServerEvent
		var recordedFreshness time.Duration
		rec := &mockRecordWorker{
			recordFn: func(_ context.Context, event *domain.ServerEvent, freshness time.Duration) error {
				recorded = event
				recordedFreshness = freshness
				return nil
			},
		}

		newService(store, rec).markUnknown(t.Context(), task)

		if recorded == nil {
			t.Fatal("expected an event to be recorded")
		}
		if recorded.Status != domain.StatusUnknown {
			t.Errorf("status = %q, want %q", recorded.Status, domain.StatusUnknown)
		}
		if recorded.EndpointID != endpointID {
			t.Errorf("endpointID = %d, want %d", recorded.EndpointID, endpointID)
		}
		if recorded.ID == (domain.ServerEvent{}).ID {
			t.Error("expected event ID to be generated")
		}
		if recordedFreshness != PushStaleInterval {
			t.Errorf("freshness = %v, want %v", recordedFreshness, PushStaleInterval)
		}
		if len(store.removed) != 1 || store.removed[0] != endpointID {
			t.Errorf("removed = %v, want [%d]", store.removed, endpointID)
		}
	})

	t.Run("record failure keeps the entry for retry", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		store := &fakeStaleStore{}
		rec := &mockRecordWorker{
			recordFn: func(_ context.Context, _ *domain.ServerEvent, _ time.Duration) error {
				return errors.New("grpc error")
			},
		}
		svc := &StaleLoopService{logger: log, freshness: store, recorder: rec}

		svc.markUnknown(t.Context(), task)

		if len(store.removed) != 0 {
			t.Errorf("entry must not be removed on record failure, got %v", store.removed)
		}
		if !capLog.HasWarn() {
			t.Error("expected warn log")
		}
	})

	t.Run("remove failure is logged but harmless", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		store := &fakeStaleStore{removeErr: errors.New("redis error")}
		rec := &mockRecordWorker{
			recordFn: func(_ context.Context, _ *domain.ServerEvent, _ time.Duration) error {
				return nil
			},
		}
		svc := &StaleLoopService{logger: log, freshness: store, recorder: rec}

		svc.markUnknown(t.Context(), task)

		if !capLog.HasError() {
			t.Error("expected error log for remove failure")
		}
	})
}
