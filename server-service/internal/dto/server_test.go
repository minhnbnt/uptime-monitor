package dto

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
)

func TestEndpointFromDomain(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		if got := EndpointFromDomain(nil); got != nil {
			t.Errorf("got %+v, want nil", got)
		}
	})

	t.Run("maps fields correctly", func(t *testing.T) {
		ep := &domain.Endpoint{
			URL:          "https://example.com",
			Interval:     30 * time.Second,
			Timeout:      10 * time.Second,
			Method:       "GET",
			ExpectedCode: 200,
		}
		got := EndpointFromDomain(ep)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if got.URL != "https://example.com" {
			t.Errorf("URL = %q, want %q", got.URL, "https://example.com")
		}
		if got.Interval != 30*time.Second {
			t.Errorf("Interval = %v, want %v", got.Interval, 30*time.Second)
		}
		if got.Timeout != 10*time.Second {
			t.Errorf("Timeout = %v, want %v", got.Timeout, 10*time.Second)
		}
		if got.Method != "GET" {
			t.Errorf("Method = %q, want %q", got.Method, "GET")
		}
		if got.ExpectedCode != 200 {
			t.Errorf("ExpectedCode = %d, want %d", got.ExpectedCode, 200)
		}
	})
}

func TestServerFromDomain(t *testing.T) {
	now := time.Now()

	t.Run("server with endpoint", func(t *testing.T) {
		srv := domain.Server{
			Model: gorm.Model{
				ID:        42,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: "Test Server",
			Endpoint: &domain.Endpoint{
				URL:          "https://example.com",
				Interval:     30 * time.Second,
				Timeout:      10 * time.Second,
				Method:       "GET",
				ExpectedCode: 200,
			},
			CreatedByID: 1,
		}

		got := ServerFromDomain(srv)
		if got.ID != 42 {
			t.Errorf("ID = %d, want %d", got.ID, 42)
		}
		if got.Name != "Test Server" {
			t.Errorf("Name = %q, want %q", got.Name, "Test Server")
		}
		if got.Endpoint == nil {
			t.Fatal("expected non-nil Endpoint")
		}
		if got.Endpoint.URL != "https://example.com" {
			t.Errorf("Endpoint.URL = %q", got.Endpoint.URL)
		}
		if !got.CreatedAt.Equal(now) {
			t.Errorf("CreatedAt mismatch")
		}
	})

	t.Run("server without endpoint", func(t *testing.T) {
		srv := domain.Server{
			Model: gorm.Model{
				ID: 1,
			},
			Name: "No Endpoint Server",
		}

		got := ServerFromDomain(srv)
		if got.Endpoint != nil {
			t.Errorf("expected nil Endpoint, got %+v", got.Endpoint)
		}
	})

	t.Run("maps timestamps correctly", func(t *testing.T) {
		createdAt := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
		updatedAt := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)

		srv := domain.Server{
			Model: gorm.Model{
				ID:        1,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			Name: "Timestamp Test",
		}

		got := ServerFromDomain(srv)
		if !got.CreatedAt.Equal(createdAt) {
			t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, createdAt)
		}
		if !got.UpdatedAt.Equal(updatedAt) {
			t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, updatedAt)
		}
	})
}
