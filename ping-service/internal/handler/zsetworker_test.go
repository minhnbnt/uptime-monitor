package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
	"gorm.io/gorm"
)

func TestPingAndRecordEndpoint(t *testing.T) {
	ep := &domain.Endpoint{
		Model:        gorm.Model{ID: 1},
		URL:          "https://example.com",
		Method:       "GET",
		ExpectedCode: 200,
	}

	t.Run("successful ping with expected code sets StatusOn", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		r := &ZSetWorkerRunner{
			pingService: &mockPingService{
				pingFn: func(_ context.Context, _ *domain.Endpoint) (bool, error) {
					return true, nil
				},
				recordFn: func(_ context.Context, event *domain.ServerEvent) error {
					recordedEvent = event
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		r.pingAndRecordEndpoint(context.Background(), ep)
		if recordedEvent == nil {
			t.Fatal("expected event to be recorded")
		}
		if recordedEvent.Status != domain.StatusOn {
			t.Errorf("status = %q, want %q", recordedEvent.Status, domain.StatusOn)
		}
		if recordedEvent.EndpointID != 1 {
			t.Errorf("endpointID = %d, want 1", recordedEvent.EndpointID)
		}
	})

	t.Run("ping error sets StatusOff", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		log, capLog := logger.NewCapturingLogger()
		r := &ZSetWorkerRunner{
			pingService: &mockPingService{
				pingFn: func(_ context.Context, _ *domain.Endpoint) (bool, error) {
					return false, errors.New("connection refused")
				},
				recordFn: func(_ context.Context, event *domain.ServerEvent) error {
					recordedEvent = event
					return nil
				},
			},
			logger: log,
		}

		r.pingAndRecordEndpoint(context.Background(), ep)
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
		r := &ZSetWorkerRunner{
			pingService: &mockPingService{
				pingFn: func(_ context.Context, _ *domain.Endpoint) (bool, error) {
					return false, nil
				},
				recordFn: func(_ context.Context, event *domain.ServerEvent) error {
					recordedEvent = event
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		r.pingAndRecordEndpoint(context.Background(), ep)
		if recordedEvent == nil {
			t.Fatal("expected event to be recorded")
		}
		if recordedEvent.Status != domain.StatusOff {
			t.Errorf("status = %q, want %q", recordedEvent.Status, domain.StatusOff)
		}
	})

	t.Run("record error is logged but not returned", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		r := &ZSetWorkerRunner{
			pingService: &mockPingService{
				pingFn: func(_ context.Context, _ *domain.Endpoint) (bool, error) {
					return true, nil
				},
				recordFn: func(_ context.Context, event *domain.ServerEvent) error {
					return errors.New("grpc error")
				},
			},
			logger: log,
		}

		r.pingAndRecordEndpoint(context.Background(), ep)
		if !capLog.HasError() {
			t.Error("expected error log for record failure")
		}
	})
}
