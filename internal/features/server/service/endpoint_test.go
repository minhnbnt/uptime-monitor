package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

func TestEndpointService_TestEndpoint(t *testing.T) {

	t.Run("success matching code", func(t *testing.T) {
		svc := &EndpointService{
			logger: logger.NewMockLogger(),
			pingWorker: &mockPinger{
				pingFn: func(_ context.Context, method, url string) (int, error) {
					if method != "GET" || url != "https://example.com/health" {
						t.Errorf("Ping(%q, %q)", method, url)
					}
					return 200, nil
				},
			},
		}

		req := dto.TestEndpointRequest{
			URL:          "https://example.com/health",
			Method:       "GET",
			Timeout:      5 * time.Second,
			ExpectedCode: 200,
		}
		resp, err := svc.TestEndpoint(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.Success {
			t.Error("expected success=true")
		}
		if resp.StatusCode != 200 {
			t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
		}
		if resp.Error != nil {
			t.Errorf("unexpected error: %v", *resp.Error)
		}
	})

	t.Run("non-matching code", func(t *testing.T) {
		svc := &EndpointService{
			logger: logger.NewMockLogger(),
			pingWorker: &mockPinger{
				pingFn: func(_ context.Context, _, _ string) (int, error) {
					return 200, nil
				},
			},
		}

		req := dto.TestEndpointRequest{
			URL:          "https://example.com/health",
			Method:       "GET",
			Timeout:      5 * time.Second,
			ExpectedCode: 404,
		}
		resp, err := svc.TestEndpoint(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Success {
			t.Error("expected success=false")
		}
		if resp.StatusCode != 200 {
			t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
		}
	})

	t.Run("ping error", func(t *testing.T) {
		svc := &EndpointService{
			logger: logger.NewMockLogger(),
			pingWorker: &mockPinger{
				pingFn: func(_ context.Context, _, _ string) (int, error) {
					return 0, errors.New("connection refused")
				},
			},
		}

		req := dto.TestEndpointRequest{
			URL:          "https://example.com/health",
			Method:       "GET",
			Timeout:      5 * time.Second,
			ExpectedCode: 200,
		}
		resp, err := svc.TestEndpoint(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Success {
			t.Error("expected success=false")
		}
		if resp.StatusCode != 0 {
			t.Errorf("StatusCode = %d, want 0", resp.StatusCode)
		}
		if resp.Error == nil || *resp.Error != "connection refused" {
			t.Errorf("Error = %v, want 'connection refused'", resp.Error)
		}
	})
}

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
		if captured.MonitorStatus != domain.StatusOff {
			t.Errorf("MonitorStatus = %q, want OFF", captured.MonitorStatus)
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
