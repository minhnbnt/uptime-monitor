package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/service/httpcheck"
)

type mockPingWorker struct {
	checkObjectStatusFn func(ctx context.Context, params *dto.K8sObjectCheckParams) (bool, error)
}

func (m *mockPingWorker) CheckObjectStatus(ctx context.Context, params *dto.K8sObjectCheckParams) (bool, error) {
	return m.checkObjectStatusFn(ctx, params)
}

type mockRecordWorker struct {
	recordFn func(ctx context.Context, event *domain.ServerEvent) error
}

func (m *mockRecordWorker) Record(ctx context.Context, event *domain.ServerEvent) error {
	return m.recordFn(ctx, event)
}

type mockHTTPChecker struct {
	checkFn func(ctx context.Context, sv *domain.Server) (bool, error)
}

func (m *mockHTTPChecker) Check(ctx context.Context, sv *domain.Server) (bool, error) {
	return m.checkFn(ctx, sv)
}

func newTestPingLoop(worker pingWorker) *PingLoopService {
	return &PingLoopService{
		pingWorker: worker,
		logger:     logger.NewMockLogger(),
	}
}

func TestPingAndRecordServer(t *testing.T) {
	sv := &domain.Server{
		ID:        1,
		Namespace: "default",
		Kind:      "Pod",
		ObjectID:  "web-app",
		Interval:  30 * time.Second,
	}

	t.Run("successful ping sets StatusOn and updates score", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		var updatedScore int64
		s := newTestPingLoop(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return true, nil
			},
		})
		s.recordStatusWorker = &mockRecordWorker{
			recordFn: func(_ context.Context, event *domain.ServerEvent) error {
				recordedEvent = event
				return nil
			},
		}
		s.scoreUpdater = &mockScoreUpdater{
			updateFn: func(_ context.Context, _ uint, nextScore int64) error {
				updatedScore = nextScore
				return nil
			},
		}

		task := PingTask{Server: sv}
		s.pingAndRecordServer(t.Context(), task)
		if recordedEvent == nil {
			t.Fatal("expected event to be recorded")
		}
		if recordedEvent.Status != domain.StatusOn {
			t.Errorf("status = %q, want %q", recordedEvent.Status, domain.StatusOn)
		}
		if recordedEvent.ServerID != 1 {
			t.Errorf("serverID = %d, want 1", recordedEvent.ServerID)
		}
		if updatedScore <= 0 {
			t.Errorf("expected positive updated score, got %d", updatedScore)
		}
	})

	t.Run("ping error sets StatusOff", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		log, capLog := logger.NewCapturingLogger()
		s := newTestPingLoop(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return false, errors.New("connection refused")
			},
		})
		s.logger = log
		s.recordStatusWorker = &mockRecordWorker{
			recordFn: func(_ context.Context, event *domain.ServerEvent) error {
				recordedEvent = event
				return nil
			},
		}
		s.scoreUpdater = &mockScoreUpdater{
			updateFn: func(_ context.Context, _ uint, _ int64) error { return nil },
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
		s := newTestPingLoop(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return false, nil
			},
		})
		s.recordStatusWorker = &mockRecordWorker{
			recordFn: func(_ context.Context, event *domain.ServerEvent) error {
				recordedEvent = event
				return nil
			},
		}
		s.scoreUpdater = &mockScoreUpdater{
			updateFn: func(_ context.Context, _ uint, _ int64) error { return nil },
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
		s := newTestPingLoop(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return true, nil
			},
		})
		s.logger = log
		s.recordStatusWorker = &mockRecordWorker{
			recordFn: func(_ context.Context, _ *domain.ServerEvent) error {
				return errors.New("grpc error")
			},
		}
		s.scoreUpdater = &mockScoreUpdater{
			updateFn: func(_ context.Context, _ uint, _ int64) error { return nil },
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if !capLog.HasError() {
			t.Error("expected error log for record failure")
		}
	})

	t.Run("score update error is logged but not returned", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		s := newTestPingLoop(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return true, nil
			},
		})
		s.logger = log
		s.recordStatusWorker = &mockRecordWorker{
			recordFn: func(_ context.Context, _ *domain.ServerEvent) error { return nil },
		}
		s.scoreUpdater = &mockScoreUpdater{
			updateFn: func(_ context.Context, _ uint, _ int64) error {
				return errors.New("redis error")
			},
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if !capLog.HasError() {
			t.Error("expected error log for score update failure")
		}
	})

	t.Run("stale domain error skips event", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		s := newTestPingLoop(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return true, nil
			},
		})
		s.httpChecker = &mockHTTPChecker{checkFn: func(_ context.Context, _ *domain.Server) (bool, error) {
			return false, httpcheck.ErrStaleDomain
		}}
		sv := &domain.Server{
			ID:        1,
			Namespace: "default",
			Kind:      "Pod",
			ObjectID:  "web-app",
			Interval:  30 * time.Second,
			HTTPConfig: &domain.ServerHTTPConfig{},
		}
		s.recordStatusWorker = &mockRecordWorker{
			recordFn: func(_ context.Context, event *domain.ServerEvent) error {
				recordedEvent = event
				return nil
			},
		}
		s.scoreUpdater = &mockScoreUpdater{
			updateFn: func(_ context.Context, _ uint, _ int64) error { return nil },
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if recordedEvent != nil {
			t.Errorf("expected no event to be recorded, got %+v", recordedEvent)
		}
	})
}
