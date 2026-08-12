package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/generated/api"
	ontimedto "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
)

func TestGetServerOntime(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeService: &mockOntimeService{
				getServerWithOntimeFn: func(_ context.Context, serverID uint, _ uuid.UUID) (*ontimedto.ServerOntime, error) {
					return &ontimedto.ServerOntime{
						ServerID: serverID,
						DayStats: []ontimedto.DayStats{},
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
				getServerWithOntimeFn: func(_ context.Context, _ uint, _ uuid.UUID) (*ontimedto.ServerOntime, error) {
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

func TestGetServersOntimeBatch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeService: &mockOntimeService{
				getServersWithOntimeFn: func(_ context.Context, _ uuid.UUID, ids []uint) ([]ontimedto.ServerOntime, error) {
					out := make([]ontimedto.ServerOntime, 0, len(ids))
					for _, id := range ids {
						out = append(out, ontimedto.ServerOntime{ServerID: id, DayStats: []ontimedto.DayStats{}})
					}
					return out, nil
				},
			},
		}
		resp, err := h.GetServersOntimeBatch(t.Context(), &api.BatchOntimeRequest{Ids: []int{1, 2}})
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
				getServersWithOntimeFn: func(_ context.Context, _ uuid.UUID, _ []uint) ([]ontimedto.ServerOntime, error) {
					return nil, errors.New("boom")
				},
			},
		}
		_, err := h.GetServersOntimeBatch(t.Context(), &api.BatchOntimeRequest{Ids: []int{1}})
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
func TestToOntimeStats_MapsHasData(t *testing.T) {
	in := []ontimedto.DayStats{
		{Date: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), Result: ontimedto.DayResult{Uptime: 100, HasData: true}},
		{Date: time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC), Result: ontimedto.DayResult{Uptime: 0, HasData: false}},
	}
	out := toOntimeStats(in)
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if !out[0].HasData.IsSet() || !out[0].HasData.Value {
		t.Errorf("server day1 HasData = %v, want true", out[0].HasData)
	}
	if out[1].HasData.Value {
		t.Errorf("server day2 HasData = true, want false")
	}
}

func TestToAPIUptimeResponse_MapsHasDataPartial(t *testing.T) {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	r := &ontimedto.UptimeResponse{
		ServerID:      1,
		Uptime:        50,
		HasData:       true,
		Partial:       true,
		From:          from.Format(time.RFC3339),
		To:            to.Format(time.RFC3339),
		TotalSeconds:  3600,
		OnlineSeconds: 1800,
		Intervals: []ontimedto.IntervalResult{
			{From: from.Format(time.RFC3339), To: to.Format(time.RFC3339), Uptime: 50, HasData: true},
		},
	}

	out, err := toAPIUptimeResponse(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.HasData.Value {
		t.Error("HasData = false, want true")
	}
	if !out.Partial.Value {
		t.Error("Partial = false, want true")
	}
	if len(out.Intervals) != 1 || !out.Intervals[0].HasData.Value {
		t.Errorf("interval has_data = %+v, want true", out.Intervals)
	}
}
