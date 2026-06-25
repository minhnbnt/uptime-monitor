package handler

import (
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
	serverdto "github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

func TestToAPIServer(t *testing.T) {
	t.Run("nil input returns empty", func(t *testing.T) {
		got := ToAPIServer(nil)
		if got.ID != 0 || got.Name != "" {
			t.Errorf("got %+v, want empty", got)
		}
	})

	t.Run("server with StatusOn", func(t *testing.T) {
		s := &serverdto.Server{
			ID:            1,
			Name:          "test",
			MonitorStatus: domain.StatusOn,
			Endpoint:      &serverdto.Endpoint{URL: "http://a.com"},
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		got := ToAPIServer(s)
		if got.MonitorStatus != api.MonitorStatusOnline {
			t.Errorf("MonitorStatus = %q", got.MonitorStatus)
		}
		if got.ID != 1 || got.Name != "test" {
			t.Errorf("unexpected server: %+v", got)
		}
	})

	t.Run("server with StatusOff", func(t *testing.T) {
		s := &serverdto.Server{
			ID:            2,
			Name:          "offline",
			MonitorStatus: domain.StatusOff,
		}
		got := ToAPIServer(s)
		if got.MonitorStatus != api.MonitorStatusOffline {
			t.Errorf("MonitorStatus = %q", got.MonitorStatus)
		}
	})

	t.Run("server with unknown status", func(t *testing.T) {
		s := &serverdto.Server{
			ID:            3,
			Name:          "unknown",
			MonitorStatus: "UNKNOWN",
		}
		got := ToAPIServer(s)
		if got.MonitorStatus != "" {
			t.Errorf("MonitorStatus = %q, want empty", got.MonitorStatus)
		}
	})

	t.Run("server without endpoint", func(t *testing.T) {
		s := &serverdto.Server{
			ID:   4,
			Name: "no-ep",
		}
		got := ToAPIServer(s)
		if got.Endpoint.Set {
			t.Error("expected empty endpoint")
		}
	})
}

func TestToPaginationMeta(t *testing.T) {
	meta := ToPaginationMeta(2, 10, 25)
	if meta.Page.Value != 2 {
		t.Errorf("Page = %d", meta.Page.Value)
	}
	if meta.PerPage.Value != 10 {
		t.Errorf("PerPage = %d", meta.PerPage.Value)
	}
	if meta.Total.Value != 25 {
		t.Errorf("Total = %d", meta.Total.Value)
	}
}

func TestToOntimeStats(t *testing.T) {
	now := time.Now()
	stats := []dto.OntimeStats{
		{Date: now, Stats: 0.95},
		{Date: now.Add(-24 * time.Hour), Stats: 0.99},
	}
	result := ToOntimeStats(stats)
	if len(result) != 2 {
		t.Fatalf("got %d, want 2", len(result))
	}
	if result[0].Stats != 0.95 {
		t.Errorf("Stats[0] = %f", result[0].Stats)
	}
	if len(ToOntimeStats(nil)) != 0 {
		t.Error("expected empty for nil")
	}
}

func TestToAPIEndpoint(t *testing.T) {
	t.Run("nil input returns empty", func(t *testing.T) {
		got := toAPIEndpoint(nil)
		if got.Set {
			t.Error("expected empty for nil")
		}
	})

	t.Run("endpoint with StatusOn", func(t *testing.T) {
		e := &serverdto.Endpoint{
			URL:           "https://example.com",
			MonitorStatus: domain.StatusOn,
			Interval:      30 * time.Second,
			Timeout:       10 * time.Second,
			Method:        "GET",
			ExpectedCode:  200,
		}
		got := toAPIEndpoint(e)
		if !got.Set {
			t.Fatal("expected set")
		}
		if got.Value.MonitorStatus.Value != api.MonitorStatusOnline {
			t.Errorf("MonitorStatus = %v", got.Value.MonitorStatus.Value)
		}
		if got.Value.Interval != 30 {
			t.Errorf("Interval = %d", got.Value.Interval)
		}
	})

	t.Run("endpoint with StatusOff", func(t *testing.T) {
		e := &serverdto.Endpoint{
			URL:           "https://example.com",
			MonitorStatus: domain.StatusOff,
		}
		got := toAPIEndpoint(e)
		if got.Value.MonitorStatus.Value != api.MonitorStatusOffline {
			t.Errorf("MonitorStatus = %v", got.Value.MonitorStatus.Value)
		}
	})
}
