package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

type mockPingWorker struct {
	checkPodStatusFn func(ctx context.Context, namespace, kind, objectID, containerName string) (bool, error)
}

func (m *mockPingWorker) CheckPodStatus(ctx context.Context, namespace, kind, objectID, containerName string) (bool, error) {
	return m.checkPodStatusFn(ctx, namespace, kind, objectID, containerName)
}

type mockRecordWorker struct {
	recordFn func(ctx context.Context, event *domain.ServerEvent) error
}

func (m *mockRecordWorker) Record(ctx context.Context, event *domain.ServerEvent) error {
	return m.recordFn(ctx, event)
}

func TestPingAndRecordServer(t *testing.T) {
	sv := &domain.Server{
		Model:     gorm.Model{ID: 1},
		ServerID:  10,
		Namespace: "default",
		Kind:      "Pod",
		ObjectID:  "web-app",
		Interval:  30 * time.Second,
	}

	t.Run("successful ping sets StatusOn and updates score", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		var updatedScore int64
		s := &PingLoopService{
			pingWorker: &mockPingWorker{
				checkPodStatusFn: func(_ context.Context, _, _, _, _ string) (bool, error) {
					return true, nil
				},
			},
			recordStatusWorker: &mockRecordWorker{
				recordFn: func(_ context.Context, event *domain.ServerEvent) error {
					recordedEvent = event
					return nil
				},
			},
			scoreUpdater: &mockScoreUpdater{
				updateFn: func(_ context.Context, _ uint, nextScore int64) error {
					updatedScore = nextScore
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		task := PingTask{Server: sv}
		s.pingAndRecordServer(t.Context(), task)
		if recordedEvent == nil {
			t.Fatal("expected event to be recorded")
		}
		if recordedEvent.Status != domain.StatusOn {
			t.Errorf("status = %q, want %q", recordedEvent.Status, domain.StatusOn)
		}
		if recordedEvent.ServerID != 10 {
			t.Errorf("serverID = %d, want 10", recordedEvent.ServerID)
		}
		if updatedScore <= 0 {
			t.Errorf("expected positive updated score, got %d", updatedScore)
		}
	})

	t.Run("ping error sets StatusOff", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		log, capLog := logger.NewCapturingLogger()
		s := &PingLoopService{
			pingWorker: &mockPingWorker{
				checkPodStatusFn: func(_ context.Context, _, _, _, _ string) (bool, error) {
					return false, errors.New("connection refused")
				},
			},
			recordStatusWorker: &mockRecordWorker{
				recordFn: func(_ context.Context, event *domain.ServerEvent) error {
					recordedEvent = event
					return nil
				},
			},
			scoreUpdater: &mockScoreUpdater{
				updateFn: func(_ context.Context, _ uint, _ int64) error { return nil },
			},
			logger: log,
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if recordedEvent == nil {
			t.Fatal("expected event to be recorded")
		}
		if recordedEvent.Status != domain.StatusOff {
			t.Errorf("status = %q, want %q", recordedEvent.Status, domain.StatusOff)
		}
		if !capLog.HasWarn() {
			t.Error("expected warn log for ping failure")
		}
	})

	t.Run("pod not running sets StatusOff", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		s := &PingLoopService{
			pingWorker: &mockPingWorker{
				checkPodStatusFn: func(_ context.Context, _, _, _, _ string) (bool, error) {
					return false, nil
				},
			},
			recordStatusWorker: &mockRecordWorker{
				recordFn: func(_ context.Context, event *domain.ServerEvent) error {
					recordedEvent = event
					return nil
				},
			},
			scoreUpdater: &mockScoreUpdater{
				updateFn: func(_ context.Context, _ uint, _ int64) error { return nil },
			},
			logger: logger.NewMockLogger(),
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if recordedEvent == nil {
			t.Fatal("expected event to be recorded")
		}
		if recordedEvent.Status != domain.StatusOff {
			t.Errorf("status = %q, want %q", recordedEvent.Status, domain.StatusOff)
		}
	})

	t.Run("record error is logged but not returned", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		s := &PingLoopService{
			pingWorker: &mockPingWorker{
				checkPodStatusFn: func(_ context.Context, _, _, _, _ string) (bool, error) {
					return true, nil
				},
			},
			recordStatusWorker: &mockRecordWorker{
				recordFn: func(_ context.Context, _ *domain.ServerEvent) error {
					return errors.New("grpc error")
				},
			},
			scoreUpdater: &mockScoreUpdater{
				updateFn: func(_ context.Context, _ uint, _ int64) error { return nil },
			},
			logger: log,
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if !capLog.HasError() {
			t.Error("expected error log for record failure")
		}
	})

	t.Run("score update error is logged but not returned", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		s := &PingLoopService{
			pingWorker: &mockPingWorker{
				checkPodStatusFn: func(_ context.Context, _, _, _, _ string) (bool, error) {
					return true, nil
				},
			},
			recordStatusWorker: &mockRecordWorker{
				recordFn: func(_ context.Context, _ *domain.ServerEvent) error { return nil },
			},
			scoreUpdater: &mockScoreUpdater{
				updateFn: func(_ context.Context, _ uint, _ int64) error {
					return errors.New("redis error")
				},
			},
			logger: log,
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if !capLog.HasError() {
			t.Error("expected error log for score update failure")
		}
	})
}
