package infrastructure

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

func newEvent(serverID uint, status domain.ServerStatus) *domain.ServerEvent {
	return &domain.ServerEvent{
		ID:       uuid.New(),
		ServerID: serverID,
		Status:   status,
	}
}

func TestRecord(t *testing.T) {
	serverID := uint(42)

	t.Run("get status error logs warn and returns nil", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return "", errors.New("redis error")
				},
			},
			eventRecorder: &mockEventRecorder{},
			logger:        log,
		}

		err := w.Record(context.Background(), newEvent(serverID, domain.StatusOn))
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
			logger: logger.NewMockLogger(),
		}

		err := w.Record(context.Background(), newEvent(serverID, domain.StatusOn))
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
		var recordedServerID uint
		var recordedStatus domain.ServerStatus
		var setServerID uint
		var setStatus domain.ServerStatus

		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return domain.StatusOff, nil
				},
				setStatusFn: func(_ context.Context, serverID uint, status domain.ServerStatus) error {
					setServerID = serverID
					setStatus = status
					return nil
				},
			},
			eventRecorder: &mockEventRecorder{
				recordEventFn: func(_ context.Context, serverID uint, status domain.ServerStatus) error {
					recordedServerID = serverID
					recordedStatus = status
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		err := w.Record(context.Background(), newEvent(serverID, domain.StatusOn))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if recordedServerID != serverID {
			t.Errorf("recorded server %d, want %d", recordedServerID, serverID)
		}
		if recordedStatus != domain.StatusOn {
			t.Errorf("recorded status %q, want %q", recordedStatus, domain.StatusOn)
		}
		if setServerID != serverID {
			t.Errorf("set server %d, want %d", setServerID, serverID)
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
			logger: logger.NewMockLogger(),
		}

		err := w.Record(context.Background(), newEvent(serverID, domain.StatusOn))
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
			logger: logger.NewMockLogger(),
		}

		err := w.Record(context.Background(), newEvent(serverID, domain.StatusOn))
		if err != wantErr {
			t.Errorf("got %v, want %v", err, wantErr)
		}
	})
}
