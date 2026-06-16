package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

func TestEndpointService_SetCheckMethod(t *testing.T) {
	req := dto.SetCheckMethodRequest{
		URL:          "https://example.com/health",
		Method:       dto.CheckMethodPull,
		HTTPMethod:   "GET",
		Interval:     60 * time.Second,
		Timeout:      10 * time.Second,
		ExpectedCode: 200,
	}

	t.Run("success", func(t *testing.T) {
		var captured *domain.Endpoint
		svc := &EndpointService{logger: logger.NewMockLogger(),
			endpointRepository: &mockEndpointRepo{
				upsertEndpointFn: func(_ context.Context, e domain.Endpoint) error {
					captured = &e
					return nil
				},
			},
		}

		err := svc.SetCheckMethod(t.Context(), 7, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if captured == nil {
			t.Fatal("endpoint not captured")
		}
		if captured.ServerID != 7 {
			t.Errorf("ServerID = %d, want 7", captured.ServerID)
		}
		if captured.Status != domain.StatusActive {
			t.Errorf("Status = %q, want active", captured.Status)
		}
		if captured.URL != req.URL {
			t.Errorf("URL = %q", captured.URL)
		}
		if captured.Interval != req.Interval {
			t.Errorf("Interval mismatch")
		}
		if captured.Timeout != req.Timeout {
			t.Errorf("Timeout mismatch")
		}
		if captured.Method != req.HTTPMethod {
			t.Errorf("Method = %q", captured.Method)
		}
		if captured.ExpectedCode != req.ExpectedCode {
			t.Errorf("ExpectedCode = %d", captured.ExpectedCode)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		svc := &EndpointService{logger: logger.NewMockLogger(),
			endpointRepository: &mockEndpointRepo{
				upsertEndpointFn: func(_ context.Context, _ domain.Endpoint) error {
					return errors.New("upsert failed")
				},
			},
		}

		err := svc.SetCheckMethod(t.Context(), 1, req)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
