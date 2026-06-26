package handler

import (
	"context"
	"io"
	"testing"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

func TestPingAndRecordEndpoint_PingOK_CodeMatch(t *testing.T) {
	var recordedEvent *domain.ServerEvent
	r := &ZSetWorkerRunner{
		pingService: &mockPingService{
			pingFn: func(_ context.Context, method, url string) (int, error) {
				return 200, nil
			},
			recordFn: func(_ context.Context, event *domain.ServerEvent) error {
				recordedEvent = event
				return nil
			},
		},
		logger: logger.NewMockLogger(),
	}

	ep := &domain.Endpoint{Method: "GET", URL: "https://example.com", ExpectedCode: 200}
	ep.ID = 1
	r.pingAndRecordEndpoint(t.Context(), ep)

	if recordedEvent == nil {
		t.Fatal("expected Record to be called")
	}
	if recordedEvent.EndpointID != 1 {
		t.Errorf("EndpointID = %d, want 1", recordedEvent.EndpointID)
	}
	if recordedEvent.Status != domain.StatusOn {
		t.Errorf("Status = %s, want %s", recordedEvent.Status, domain.StatusOn)
	}
}

func TestPingAndRecordEndpoint_PingOK_CodeMismatch(t *testing.T) {
	var recordedEvent *domain.ServerEvent
	r := &ZSetWorkerRunner{
		pingService: &mockPingService{
			pingFn: func(_ context.Context, method, url string) (int, error) {
				return 200, nil
			},
			recordFn: func(_ context.Context, event *domain.ServerEvent) error {
				recordedEvent = event
				return nil
			},
		},
		logger: logger.NewMockLogger(),
	}

	ep := &domain.Endpoint{Method: "GET", URL: "https://example.com", ExpectedCode: 201}
	ep.ID = 2
	r.pingAndRecordEndpoint(t.Context(), ep)

	if recordedEvent == nil {
		t.Fatal("expected Record to be called")
	}
	if recordedEvent.Status != domain.StatusOff {
		t.Errorf("Status = %s, want %s", recordedEvent.Status, domain.StatusOff)
	}
}

func TestPingAndRecordEndpoint_PingError(t *testing.T) {
	var recordedEvent *domain.ServerEvent
	mockLog := logger.NewMockLogger()
	r := &ZSetWorkerRunner{
		pingService: &mockPingService{
			pingFn: func(_ context.Context, method, url string) (int, error) {
				return 0, io.ErrUnexpectedEOF
			},
			recordFn: func(_ context.Context, event *domain.ServerEvent) error {
				recordedEvent = event
				return nil
			},
		},
		logger: mockLog,
	}

	ep := &domain.Endpoint{Method: "GET", URL: "https://example.com", ExpectedCode: 200}
	ep.ID = 3
	r.pingAndRecordEndpoint(t.Context(), ep)

	if recordedEvent == nil {
		t.Fatal("expected Record to be called")
	}
	if recordedEvent.Status != domain.StatusOff {
		t.Errorf("Status = %s, want %s", recordedEvent.Status, domain.StatusOff)
	}
	if !mockLog.ErrorCalled {
		t.Error("expected error log for ping failure")
	}
}

func TestPingAndRecordEndpoint_RecordError(t *testing.T) {
	mockLog := logger.NewMockLogger()
	r := &ZSetWorkerRunner{
		pingService: &mockPingService{
			pingFn: func(_ context.Context, method, url string) (int, error) {
				return 200, nil
			},
			recordFn: func(_ context.Context, event *domain.ServerEvent) error {
				return io.ErrClosedPipe
			},
		},
		logger: mockLog,
	}

	ep := &domain.Endpoint{Method: "GET", URL: "https://example.com", ExpectedCode: 200}
	ep.ID = 4
	r.pingAndRecordEndpoint(t.Context(), ep)

	if !mockLog.ErrorCalled {
		t.Error("expected error log for record failure")
	}
}
