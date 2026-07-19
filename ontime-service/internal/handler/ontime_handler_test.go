package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

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
