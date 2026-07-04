package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	ontimedto "github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

func dtoServer(id uint, name string, t time.Time) dto.Server {
	return dto.Server{ID: id, Name: name, CreatedAt: t, UpdatedAt: t}
}

func dtoOntimeStats(date time.Time, stats float64) ontimedto.OntimeStats {
	return ontimedto.OntimeStats{Date: date, Stats: stats}
}

func TestOntimeHandler_GetServer(t *testing.T) {
	now := time.Now()
	stats := []ontimedto.OntimeStats{
		dtoOntimeStats(now, 99.5),
		dtoOntimeStats(now.Add(-24*time.Hour), 100.0),
	}

	t.Run("success", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeService: &mockOntimeService{
				getServerWithOntimeFn: func(_ context.Context, id uint, _ uint) (*ontimedto.ServerWithOntime, error) {
					return &ontimedto.ServerWithOntime{
						Server:      dtoServer(id, "server-a", now),
						OntimeStats: stats,
					}, nil
				},
			},
		}
		resp, err := h.GetServer(context.Background(), api.GetServerParams{ID: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Data.Name != "server-a" {
			t.Errorf("name = %q, want server-a", resp.Data.Name)
		}
		if len(resp.Data.OntimeStats) != 2 {
			t.Errorf("len stats = %d, want 2", len(resp.Data.OntimeStats))
		}
	})

	t.Run("not found", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeService: &mockOntimeService{
				getServerWithOntimeFn: func(_ context.Context, _ uint, _ uint) (*ontimedto.ServerWithOntime, error) {
					return nil, apperrors.ErrNotFound
				},
			},
		}
		_, err := h.GetServer(context.Background(), api.GetServerParams{ID: 99})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeService: &mockOntimeService{
				getServerWithOntimeFn: func(_ context.Context, _ uint, _ uint) (*ontimedto.ServerWithOntime, error) {
					return nil, errors.New("db error")
				},
			},
		}
		_, err := h.GetServer(context.Background(), api.GetServerParams{ID: 1})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusInternalServerError)
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeService: &mockOntimeService{
				getServerWithOntimeFn: func(_ context.Context, _ uint, _ uint) (*ontimedto.ServerWithOntime, error) {
					return nil, apperrors.ErrForbidden
				},
			},
		}
		_, err := h.GetServer(context.Background(), api.GetServerParams{ID: 1})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusForbidden {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusForbidden)
		}
	})
}

func TestOntimeHandler_ListServersOntime(t *testing.T) {
	now := time.Now()
	stats := []ontimedto.OntimeStats{
		dtoOntimeStats(now, 100.0),
	}

	t.Run("success", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeService: &mockOntimeService{
				listServersWithOntimeFn: func(_ context.Context, createdByID uint, page, perPage int) ([]ontimedto.ServerWithOntime, int64, int64, int64, error) {
					return []ontimedto.ServerWithOntime{
						{Server: dtoServer(1, "s1", now), OntimeStats: stats},
						{Server: dtoServer(2, "s2", now), OntimeStats: stats},
					}, 5, 2, 3, nil
				},
			},
		}
		resp, err := h.ListServersOntime(context.Background(), api.ListServersOntimeParams{
			Page:    api.NewOptInt(1),
			PerPage: api.NewOptInt(10),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Data) != 2 {
			t.Errorf("len data = %d, want 2", len(resp.Data))
		}
		if resp.Data[0].Server.Name != "s1" {
			t.Errorf("name = %q, want s1", resp.Data[0].Server.Name)
		}
		if resp.Meta.Total.Or(0) != 5 {
			t.Errorf("total = %d, want 5", resp.Meta.Total.Or(0))
		}
		if resp.OnlineCount != 2 {
			t.Errorf("online_count = %d, want 2", resp.OnlineCount)
		}
		if resp.OfflineCount != 3 {
			t.Errorf("offline_count = %d, want 3", resp.OfflineCount)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &OntimeHandler{
			ontimeService: &mockOntimeService{
				listServersWithOntimeFn: func(_ context.Context, _ uint, _, _ int) ([]ontimedto.ServerWithOntime, int64, int64, int64, error) {
					return nil, 0, 0, 0, errors.New("db error")
				},
			},
		}
		_, err := h.ListServersOntime(context.Background(), api.ListServersOntimeParams{})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusInternalServerError)
		}
	})
}
