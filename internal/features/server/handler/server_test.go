package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

func dtoServer(id uint, name string, t time.Time) dto.Server {
	return dto.Server{ID: id, Name: name, CreatedAt: t, UpdatedAt: t}
}

func TestServerHandler_ListServers(t *testing.T) {

	now := time.Now()

	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				listServersFn: func(_ context.Context, createdByID uint, page, perPage int) ([]dto.Server, int64, error) {
					if page != 2 || perPage != 10 {
						t.Errorf("ListServers(%d, %d)", page, perPage)
					}
					return []dto.Server{dtoServer(1, "s1", now)}, 5, nil
				},
			},
		}

		resp, err := h.ListServers(t.Context(), api.ListServersParams{
			Page:    api.NewOptInt(2),
			PerPage: api.NewOptInt(10),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Meta.Total.Or(0) != 5 {
			t.Errorf("total = %d, want 5", resp.Meta.Total.Or(0))
		}
		if len(resp.Data) != 1 || resp.Data[0].Name != "s1" {
			t.Errorf("unexpected data: %+v", resp.Data)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				listServersFn: func(_ context.Context, _ uint, _, _ int) ([]dto.Server, int64, error) {
					return nil, 0, errors.New("db error")
				},
			},
		}
		_, err := h.ListServers(t.Context(), api.ListServersParams{})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusInternalServerError)
		}
	})
}

func TestServerHandler_CreateServer(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				createServerFn: func(_ context.Context, req dto.CreateServerRequest, createdByID uint) (*dto.Server, error) {
					s := dtoServer(1, req.Name, now)
					return &s, nil
				},
			},
		}
		req := &api.CreateServerRequest{Name: "new-srv"}
		resp, err := h.CreateServer(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Data.Name != "new-srv" {
			t.Errorf("name = %q", resp.Data.Name)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				createServerFn: func(_ context.Context, _ dto.CreateServerRequest, _ uint) (*dto.Server, error) {
					return nil, errors.New("db error")
				},
			},
		}
		req := &api.CreateServerRequest{Name: "x"}
		_, err := h.CreateServer(t.Context(), req)
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusInternalServerError)
		}
	})
}

func TestServerHandler_UpdateServer(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				updateServerFn: func(_ context.Context, id uint, _ uint, req dto.UpdateServerRequest) (*dto.Server, error) {
					s := dtoServer(id, *req.Name, now)
					return &s, nil
				},
			},
		}
		req := &api.UpdateServerRequest{Name: api.NewOptString("updated")}
		resp, err := h.UpdateServer(t.Context(), req, api.UpdateServerParams{ID: 3})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Data.Name != "updated" {
			t.Errorf("name = %q", resp.Data.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				updateServerFn: func(_ context.Context, _ uint, _ uint, _ dto.UpdateServerRequest) (*dto.Server, error) {
					return nil, apperrors.ErrNotFound
				},
			},
		}
		req := &api.UpdateServerRequest{Name: api.NewOptString("x")}
		_, err := h.UpdateServer(t.Context(), req, api.UpdateServerParams{ID: 99})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				updateServerFn: func(_ context.Context, _ uint, _ uint, _ dto.UpdateServerRequest) (*dto.Server, error) {
					return nil, apperrors.ErrForbidden
				},
			},
		}
		req := &api.UpdateServerRequest{Name: api.NewOptString("x")}
		_, err := h.UpdateServer(t.Context(), req, api.UpdateServerParams{ID: 1})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusForbidden {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusForbidden)
		}
	})
}

func TestServerHandler_DeleteServer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				deleteServerFn: func(_ context.Context, id uint, _ uint) error {
					return nil
				},
			},
		}
		err := h.DeleteServer(t.Context(), api.DeleteServerParams{ID: 4})
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				deleteServerFn: func(_ context.Context, _ uint, _ uint) error {
					return apperrors.ErrNotFound
				},
			},
		}
		err := h.DeleteServer(t.Context(), api.DeleteServerParams{ID: 99})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				deleteServerFn: func(_ context.Context, _ uint, _ uint) error {
					return apperrors.ErrForbidden
				},
			},
		}
		err := h.DeleteServer(t.Context(), api.DeleteServerParams{ID: 1})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusForbidden {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusForbidden)
		}
	})
}
