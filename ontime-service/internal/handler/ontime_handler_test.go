package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/generated/api"
	ontimedto "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
)

func TestGetServerOntime(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeService: &mockOntimeService{
				getServerWithOntimeFn: func(_ context.Context, serverID, _ uint) (*ontimedto.ServerOntime, error) {
					return &ontimedto.ServerOntime{
						ServerID:    serverID,
						OntimeStats: []ontimedto.OntimeStats{},
					}, nil
				},
			},
		}
		resp, err := h.GetServerOntime(t.Context(), api.GetServerOntimeParams{ID: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.Data.IsSet() {
			t.Fatal("expected data")
		}
		if resp.Data.Value.ServerID.Value != 1 {
			t.Errorf("server_id = %d, want 1", resp.Data.Value.ServerID.Value)
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeService: &mockOntimeService{
				getServerWithOntimeFn: func(_ context.Context, _, _ uint) (*ontimedto.ServerOntime, error) {
					return nil, errors.New("some error")
				},
			},
		}
		_, err := h.GetServerOntime(t.Context(), api.GetServerOntimeParams{ID: 1})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestListServersOntime(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeService: &mockOntimeService{
				listServersWithOntimeFn: func(_ context.Context, _ uint, _, _ int) ([]ontimedto.ServerOntime, error) {
					return []ontimedto.ServerOntime{
						{ServerID: 1, OntimeStats: []ontimedto.OntimeStats{}},
						{ServerID: 2, OntimeStats: []ontimedto.OntimeStats{}},
					}, nil
				},
			},
		}
		resp, err := h.ListServersOntime(t.Context(), api.ListServersOntimeParams{
			Page:    api.NewOptInt(1),
			PerPage: api.NewOptInt(10),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Data) != 2 {
			t.Errorf("len data = %d, want 2", len(resp.Data))
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeService: &mockOntimeService{
				listServersWithOntimeFn: func(_ context.Context, _ uint, _, _ int) ([]ontimedto.ServerOntime, error) {
					return nil, errors.New("db error")
				},
			},
		}
		_, err := h.ListServersOntime(t.Context(), api.ListServersOntimeParams{})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestNewError(t *testing.T) {
	h := &OntimeHandler{}

	t.Run("not found", func(t *testing.T) {
		err := h.NewError(t.Context(), apperrors.ErrNotFound)
		if err.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want %d", err.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		err := h.NewError(t.Context(), apperrors.ErrForbidden)
		if err.StatusCode != http.StatusForbidden {
			t.Errorf("status = %d, want %d", err.StatusCode, http.StatusForbidden)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		err := h.NewError(t.Context(), errors.New("db error"))
		if err.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", err.StatusCode, http.StatusInternalServerError)
		}
	})
}

func TestCalculateUptime(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeRangeService: &mockOntimeRangeService{
				calculateUptimeFn: func(_ context.Context, in ontimedto.CalculateUptimeInput) (*ontimedto.UptimeResponse, error) {
					return &ontimedto.UptimeResponse{
						ServerID:      in.ServerID,
						Uptime:        99.5,
						HasData:       true,
						Partial:       false,
						From:          "2026-01-01T00:00:00Z",
						To:            "2026-01-02T00:00:00Z",
						TotalSeconds:  86400,
						OnlineSeconds: 85968,
						Intervals: []ontimedto.IntervalResult{
							{From: "2026-01-01T00:00:00Z", To: "2026-01-01T00:15:00Z", Uptime: 100, HasData: true},
						},
					}, nil
				},
			},
		}
		resp, err := h.CalculateUptime(
			t.Context(),
			&api.CalculateUptimeRequest{
				From: parseTime(t, "2026-01-01T00:00:00Z"),
				To:   parseTime(t, "2026-01-02T00:00:00Z"),
			},
			api.CalculateUptimeParams{ID: 1},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.Uptime.IsSet() {
			t.Fatal("expected uptime")
		}
		if resp.Uptime.Value != 99.5 {
			t.Errorf("uptime = %f, want 99.5", resp.Uptime.Value)
		}
		if len(resp.Intervals) != 1 {
			t.Errorf("intervals len = %d, want 1", len(resp.Intervals))
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeRangeService: &mockOntimeRangeService{
				calculateUptimeFn: func(_ context.Context, _ ontimedto.CalculateUptimeInput) (*ontimedto.UptimeResponse, error) {
					return nil, errors.New("some error")
				},
			},
		}
		_, err := h.CalculateUptime(
			t.Context(),
			&api.CalculateUptimeRequest{
				From: parseTime(t, "2026-01-01T00:00:00Z"),
				To:   parseTime(t, "2026-01-02T00:00:00Z"),
			},
			api.CalculateUptimeParams{ID: 1},
		)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func parseTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse time: %v", err)
	}
	return v
}
