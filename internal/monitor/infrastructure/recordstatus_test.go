package infrastructure

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

type mockStatusStore struct {
	getStatusFn func(ctx context.Context, endpointID uint) (domain.ServerStatus, error)
	setStatusFn func(ctx context.Context, endpointID uint, status domain.ServerStatus) error
}

func (m *mockStatusStore) GetStatus(ctx context.Context, endpointID uint) (domain.ServerStatus, error) {
	return m.getStatusFn(ctx, endpointID)
}
func (m *mockStatusStore) SetStatus(ctx context.Context, endpointID uint, status domain.ServerStatus) error {
	return m.setStatusFn(ctx, endpointID, status)
}

type mockEventSaver struct {
	saveFn func(ctx context.Context, event *domain.ServerEvent) error
}

func (m *mockEventSaver) Save(ctx context.Context, event *domain.ServerEvent) error {
	return m.saveFn(ctx, event)
}

func event(endpointID uint, status domain.ServerStatus) *domain.ServerEvent {
	return &domain.ServerEvent{
		ID:         uuid.Must(uuid.NewV7()),
		EndpointID: endpointID,
		Status:     status,
	}
}

func TestRecordStatusWorker_Record(t *testing.T) {
	t.Run("redis get fails -> log warning, return nil", func(t *testing.T) {
		log := logger.NewMockLogger()
		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return "", errors.New("redis down")
				},
			},
			logger: log,
		}

		err := w.Record(t.Context(), event(1, domain.StatusOn))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !log.WarnCalled {
			t.Error("expected Warn to be called")
		}
	})

	t.Run("same status -> no-op", func(t *testing.T) {
		var saveCalled bool
		var setCalled bool
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
			eventSaver: &mockEventSaver{
				saveFn: func(_ context.Context, _ *domain.ServerEvent) error {
					saveCalled = true
					return nil
				},
			},
		}

		err := w.Record(t.Context(), event(1, domain.StatusOn))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if saveCalled {
			t.Error("Save should not be called for same status")
		}
		if setCalled {
			t.Error("SetStatus should not be called for same status")
		}
	})

	t.Run("status changed -> save and set", func(t *testing.T) {
		var savedEvent *domain.ServerEvent
		var setEndpointID uint
		var setStatus domain.ServerStatus

		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return domain.StatusOn, nil
				},
				setStatusFn: func(_ context.Context, endpointID uint, status domain.ServerStatus) error {
					setEndpointID = endpointID
					setStatus = status
					return nil
				},
			},
			eventSaver: &mockEventSaver{
				saveFn: func(_ context.Context, event *domain.ServerEvent) error {
					savedEvent = event
					return nil
				},
			},
		}

		e := event(7, domain.StatusOff)
		err := w.Record(t.Context(), e)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if savedEvent == nil {
			t.Fatal("Save was not called")
		}
		if savedEvent.EndpointID != 7 || savedEvent.Status != domain.StatusOff {
			t.Errorf("unexpected saved event: %+v", savedEvent)
		}
		if setEndpointID != 7 {
			t.Errorf("set endpointID = %d, want 7", setEndpointID)
		}
		if setStatus != domain.StatusOff {
			t.Errorf("set status = %s, want OFF", setStatus)
		}
	})

	t.Run("db save error -> return error", func(t *testing.T) {
		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return domain.StatusOn, nil
				},
			},
			eventSaver: &mockEventSaver{
				saveFn: func(_ context.Context, _ *domain.ServerEvent) error {
					return errors.New("db error")
				},
			},
		}

		err := w.Record(t.Context(), event(1, domain.StatusOff))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("redis set error -> return error", func(t *testing.T) {
		w := &RecordStatusWorker{
			statusStore: &mockStatusStore{
				getStatusFn: func(_ context.Context, _ uint) (domain.ServerStatus, error) {
					return domain.StatusOn, nil
				},
				setStatusFn: func(_ context.Context, _ uint, _ domain.ServerStatus) error {
					return errors.New("redis set error")
				},
			},
			eventSaver: &mockEventSaver{
				saveFn: func(_ context.Context, _ *domain.ServerEvent) error {
					return nil
				},
			},
		}

		err := w.Record(t.Context(), event(1, domain.StatusOff))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
