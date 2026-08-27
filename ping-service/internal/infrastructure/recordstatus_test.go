package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

const testFreshness = 90 * time.Second

type mockFreshnessToucher struct {
	touched []uint
	lease   time.Duration
	err     error
}

func (m *mockFreshnessToucher) Touch(_ context.Context, endpointID uint, lease time.Duration) error {
	m.touched = append(m.touched, endpointID)
	m.lease = lease
	return m.err
}

func newEvent(endpointID uint, status domain.ServerStatus) *domain.ServerEvent {
	return &domain.ServerEvent{
		ID:         uuid.New(),
		EndpointID: endpointID,
		Status:     status,
	}
}

func TestRecord(t *testing.T) {
	endpointID := uint(42)

	t.Run("get status error logs warn and returns nil", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return "", errors.New("redis error")
				},
			},
			eventRecorder: &mockEventRecorder{},
			freshness:     &mockFreshnessToucher{},
			logger:        log,
		}

		err := w.Record(context.Background(), newEvent(endpointID, domain.StatusOn), testFreshness)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !capLog.HasWarn() {
			t.Error("expected warn log")
		}
	})

	t.Run("same status does nothing", func(t *testing.T) {
		var recordCalled, setCalled bool
		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return domain.StatusOn, nil
				},
				setStatusFn: func(_ context.Context, _ uint, _ domain.ServerStatus) error {
					setCalled = true
					return nil
				},
			},
			eventRecorder: &mockEventRecorder{
				recordEventFn: func(_ context.Context, _ uint, _ domain.ServerStatus) error {
					recordCalled = true
					return nil
				},
			},
			freshness: &mockFreshnessToucher{},
			logger:    logger.NewMockLogger(),
		}

		err := w.Record(context.Background(), newEvent(endpointID, domain.StatusOn), testFreshness)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if recordCalled {
			t.Error("RecordEvent should not be called when status unchanged")
		}
		if setCalled {
			t.Error("SetStatus should not be called when status unchanged")
		}
	})

	t.Run("status changed records event and updates status", func(t *testing.T) {
		var recordedEndpointID uint
		var recordedStatus domain.ServerStatus
		var setEndpointID uint
		var setStatus domain.ServerStatus

		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return domain.StatusOff, nil
				},
				setStatusFn: func(_ context.Context, endpointID uint, status domain.ServerStatus) error {
					setEndpointID = endpointID
					setStatus = status
					return nil
				},
			},
			eventRecorder: &mockEventRecorder{
				recordEventFn: func(_ context.Context, endpointID uint, status domain.ServerStatus) error {
					recordedEndpointID = endpointID
					recordedStatus = status
					return nil
				},
			},
			freshness: &mockFreshnessToucher{},
			logger:    logger.NewMockLogger(),
		}

		err := w.Record(context.Background(), newEvent(endpointID, domain.StatusOn), testFreshness)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if recordedEndpointID != endpointID {
			t.Errorf("recorded endpoint %d, want %d", recordedEndpointID, endpointID)
		}
		if recordedStatus != domain.StatusOn {
			t.Errorf("recorded status %q, want %q", recordedStatus, domain.StatusOn)
		}
		if setEndpointID != endpointID {
			t.Errorf("set endpoint %d, want %d", setEndpointID, endpointID)
		}
		if setStatus != domain.StatusOn {
			t.Errorf("set status %q, want %q", setStatus, domain.StatusOn)
		}
	})

	t.Run("record event error propagates", func(t *testing.T) {
		wantErr := errors.New("grpc error")
		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return domain.StatusOff, nil
				},
			},
			eventRecorder: &mockEventRecorder{
				recordEventFn: func(_ context.Context, _ uint, _ domain.ServerStatus) error {
					return wantErr
				},
			},
			freshness: &mockFreshnessToucher{},
			logger:    logger.NewMockLogger(),
		}

		err := w.Record(context.Background(), newEvent(endpointID, domain.StatusOn), testFreshness)
		if err != wantErr {
			t.Errorf("got %v, want %v", err, wantErr)
		}
	})

	t.Run("set status error propagates after successful record", func(t *testing.T) {
		wantErr := errors.New("redis set error")
		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return domain.StatusOff, nil
				},
				setStatusFn: func(_ context.Context, _ uint, _ domain.ServerStatus) error {
					return wantErr
				},
			},
			eventRecorder: &mockEventRecorder{
				recordEventFn: func(_ context.Context, _ uint, _ domain.ServerStatus) error {
					return nil
				},
			},
			freshness: &mockFreshnessToucher{},
			logger:    logger.NewMockLogger(),
		}

		err := w.Record(context.Background(), newEvent(endpointID, domain.StatusOn), testFreshness)
		if err != wantErr {
			t.Errorf("got %v, want %v", err, wantErr)
		}
	})

	t.Run("touches freshness even when dedupe skips recording", func(t *testing.T) {
		fresh := &mockFreshnessToucher{}
		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return domain.StatusOn, nil
				},
			},
			eventRecorder: &mockEventRecorder{},
			freshness:     fresh,
			logger:        logger.NewMockLogger(),
		}

		err := w.Record(context.Background(), newEvent(endpointID, domain.StatusOn), testFreshness)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(fresh.touched) != 1 || fresh.touched[0] != endpointID {
			t.Errorf("touched = %v, want [%d]", fresh.touched, endpointID)
		}
		if fresh.lease != testFreshness {
			t.Errorf("lease = %v, want %v", fresh.lease, testFreshness)
		}
	})

	t.Run("live Record stamps ~now (historical time is the stale path's job)", func(t *testing.T) {
		var got time.Time
		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return domain.StatusOff, nil
				},
			},
			eventRecorder: &mockEventRecorder{
				recordEventAtFn: func(_ context.Context, _ uint, _ domain.ServerStatus, recordedAt time.Time) error {
					got = recordedAt
					return nil
				},
			},
			freshness: &mockFreshnessToucher{},
			logger:    logger.NewMockLogger(),
		}

		if err := w.Record(context.Background(), newEvent(endpointID, domain.StatusOn), testFreshness); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.IsZero() || got.Before(time.Now().Add(-2*time.Second)) || got.After(time.Now().Add(2*time.Second)) {
			t.Errorf("recorded time = %v, want ~now", got)
		}
	})

	t.Run("touch error logs warn but does not drop the event", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		var recorded bool
		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return domain.StatusOff, nil
				},
			},
			eventRecorder: &mockEventRecorder{
				recordEventFn: func(_ context.Context, _ uint, _ domain.ServerStatus) error {
					recorded = true
					return nil
				},
			},
			freshness: &mockFreshnessToucher{err: errors.New("redis error")},
			logger:    log,
		}

		err := w.Record(context.Background(), newEvent(endpointID, domain.StatusOn), testFreshness)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !capLog.HasWarn() {
			t.Error("expected warn log for touch failure")
		}
		if !recorded {
			t.Error("event must still be recorded when touch fails")
		}
	})
}
