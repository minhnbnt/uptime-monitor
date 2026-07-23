package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	infra "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

type mockPingWorker struct {
	pingFn func(ctx context.Context, ep *domain.Endpoint) (*infra.Response, error)
}

func (m *mockPingWorker) Ping(ctx context.Context, ep *domain.Endpoint) (*infra.Response, error) {
	return m.pingFn(ctx, ep)
}

type mockRecordWorker struct {
	recordFn func(ctx context.Context, event *domain.ServerEvent) error
}

func (m *mockRecordWorker) Record(ctx context.Context, event *domain.ServerEvent) error {
	return m.recordFn(ctx, event)
}

func TestPingAndRecordEndpoint(t *testing.T) {
	ep := &domain.Endpoint{
		Model:        gorm.Model{ID: 1},
		URL:          "https://example.com",
		Method:       "GET",
		ExpectedCode: 200,
		Interval:     30 * time.Second,
	}

	t.Run("successful ping with expected code sets StatusOn and updates score", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		var updatedScore int64
		s := &PingLoopService{
			pingWorker: &mockPingWorker{
				pingFn: func(_ context.Context, _ *domain.Endpoint) (*infra.Response, error) {
					return &infra.Response{StatusCode: 200}, nil
				},
			},
			responseChecker: &ResponseChecker{bodyChecker: &infra.BodyChecker{}},
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

		s.pingAndRecordEndpoint(t.Context(), ep)
		if recordedEvent == nil {
			t.Fatal("expected event to be recorded")
		}
		if recordedEvent.Status != domain.StatusOn {
			t.Errorf("status = %q, want %q", recordedEvent.Status, domain.StatusOn)
		}
		if recordedEvent.EndpointID != 1 {
			t.Errorf("endpointID = %d, want 1", recordedEvent.EndpointID)
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
				pingFn: func(_ context.Context, _ *domain.Endpoint) (*infra.Response, error) {
					return nil, errors.New("connection refused")
				},
			},
			responseChecker: &ResponseChecker{bodyChecker: &infra.BodyChecker{}},
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

		s.pingAndRecordEndpoint(t.Context(), ep)
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

	t.Run("status code mismatch sets StatusOff", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		s := &PingLoopService{
			pingWorker: &mockPingWorker{
				pingFn: func(_ context.Context, _ *domain.Endpoint) (*infra.Response, error) {
					return &infra.Response{StatusCode: 500}, nil
				},
			},
			responseChecker: &ResponseChecker{bodyChecker: &infra.BodyChecker{}},
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

		s.pingAndRecordEndpoint(t.Context(), ep)
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
				pingFn: func(_ context.Context, _ *domain.Endpoint) (*infra.Response, error) {
					return &infra.Response{StatusCode: 200}, nil
				},
			},
			responseChecker: &ResponseChecker{bodyChecker: &infra.BodyChecker{}},
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

		s.pingAndRecordEndpoint(t.Context(), ep)
		if !capLog.HasError() {
			t.Error("expected error log for record failure")
		}
	})

	t.Run("score update error is logged but not returned", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		s := &PingLoopService{
			pingWorker: &mockPingWorker{
				pingFn: func(_ context.Context, _ *domain.Endpoint) (*infra.Response, error) {
					return &infra.Response{StatusCode: 200}, nil
				},
			},
			responseChecker: &ResponseChecker{bodyChecker: &infra.BodyChecker{}},
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

		s.pingAndRecordEndpoint(t.Context(), ep)
		if !capLog.HasError() {
			t.Error("expected error log for score update failure")
		}
	})
}
