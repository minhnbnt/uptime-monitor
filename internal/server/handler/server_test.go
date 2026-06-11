package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

func dtoServer(id uint, name string, t time.Time) dto.Server {
	return dto.Server{ID: id, Name: name, Status: "active", CreatedAt: t, UpdatedAt: t}
}

func TestServerHandler_ListServers(t *testing.T) {

	now := time.Now()

	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				listServersFn: func(_ context.Context, createdByID uint, page, perPage int) ([]dto.Server, error) {
					if page != 2 || perPage != 10 {
						t.Errorf("ListServers(%d, %d)", page, perPage)
					}
					return []dto.Server{dtoServer(1, "s1", now)}, nil
				},
			},
		}

		resp, err := h.ListServers(context.Background(), api.ListServersParams{
			Page:    api.NewOptInt(2),
			PerPage: api.NewOptInt(10),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Data) != 1 || resp.Data[0].Name != "s1" {
			t.Errorf("unexpected data: %+v", resp.Data)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				listServersFn: func(_ context.Context, _ uint, _, _ int) ([]dto.Server, error) {
					return nil, errors.New("db error")
				},
			},
		}
		_, err := h.ListServers(context.Background(), api.ListServersParams{})
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
		resp, err := h.CreateServer(context.Background(), req)
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
		_, err := h.CreateServer(context.Background(), req)
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusInternalServerError)
		}
	})
}

func TestServerHandler_GetServer(t *testing.T) {
	now := time.Now()

	t.Run("found", func(t *testing.T) {
		h := &ServerHandler{
			ontimeService: &mockOntimeService{
				getServerWithOntimeFn: func(_ context.Context, _ uint) (*dto.ServerWithOntime, error) {
					s := dtoServer(5, "found", now)
					return &dto.ServerWithOntime{
						Server:      s,
						OntimeStats: nil,
					}, nil
				},
			},
		}
		resp, err := h.GetServer(context.Background(), api.GetServerParams{ID: 5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Data.ID != 5 || resp.Data.Name != "found" {
			t.Errorf("unexpected: %+v", resp.Data)
		}
	})

	t.Run("with ontime stats", func(t *testing.T) {
		h := &ServerHandler{
			ontimeService: &mockOntimeService{
				getServerWithOntimeFn: func(_ context.Context, _ uint) (*dto.ServerWithOntime, error) {
					s := dtoServer(5, "found", now)
					return &dto.ServerWithOntime{
						Server: s,
						OntimeStats: []dto.OntimeStats{
							{Date: now, Stats: 99.5},
						},
					}, nil
				},
			},
		}
		resp, err := h.GetServer(context.Background(), api.GetServerParams{ID: 5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Data.ID != 5 {
			t.Errorf("ID = %d, want 5", resp.Data.ID)
		}
		if len(resp.Data.OntimeStats) != 1 {
			t.Fatalf("len(OntimeStats) = %d, want 1", len(resp.Data.OntimeStats))
		}
		if resp.Data.OntimeStats[0].GetStats() != 99.5 {
			t.Errorf("stats = %f, want 99.5", resp.Data.OntimeStats[0].GetStats())
		}
	})

	t.Run("not found", func(t *testing.T) {
		h := &ServerHandler{
			ontimeService: &mockOntimeService{
				getServerWithOntimeFn: func(_ context.Context, _ uint) (*dto.ServerWithOntime, error) {
					return nil, errors.New("not found")
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
}

func TestServerHandler_UpdateServer(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				updateServerFn: func(_ context.Context, id uint, req dto.UpdateServerRequest) (*dto.Server, error) {
					s := dtoServer(id, *req.Name, now)
					return &s, nil
				},
			},
		}
		req := &api.UpdateServerRequest{Name: api.NewOptString("updated")}
		resp, err := h.UpdateServer(context.Background(), req, api.UpdateServerParams{ID: 3})
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
				updateServerFn: func(_ context.Context, _ uint, _ dto.UpdateServerRequest) (*dto.Server, error) {
					return nil, errors.New("not found")
				},
			},
		}
		req := &api.UpdateServerRequest{Name: api.NewOptString("x")}
		_, err := h.UpdateServer(context.Background(), req, api.UpdateServerParams{ID: 99})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusNotFound)
		}
	})
}

func TestServerHandler_DeleteServer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				deleteServerFn: func(_ context.Context, id uint) error {
					return nil
				},
			},
		}
		err := h.DeleteServer(context.Background(), api.DeleteServerParams{ID: 4})
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				deleteServerFn: func(_ context.Context, _ uint) error {
					return errors.New("not found")
				},
			},
		}
		err := h.DeleteServer(context.Background(), api.DeleteServerParams{ID: 99})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusNotFound)
		}
	})
}

func TestServerHandler_ListServersOntime(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			ontimeService: &mockOntimeService{
				listServersWithOntimeFn: func(_ context.Context, createdByID uint, page, perPage int) ([]dto.ServerWithOntime, int64, error) {
					return []dto.ServerWithOntime{
						{Server: dtoServer(1, "s1", now), OntimeStats: []dto.OntimeStats{{Date: now, Stats: 95.5}}},
					}, 1, nil
				},
			},
		}

		resp, err := h.ListServersOntime(context.Background(), api.ListServersOntimeParams{
			Page:    api.NewOptInt(1),
			PerPage: api.NewOptInt(20),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Data) != 1 || resp.Data[0].Server.Name != "s1" {
			t.Errorf("unexpected data: %+v", resp.Data)
		}
		total, ok := resp.Meta.Total.Get()
		if !ok || total != 1 {
			t.Errorf("total = %v", resp.Meta.Total)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &ServerHandler{
			ontimeService: &mockOntimeService{
				listServersWithOntimeFn: func(_ context.Context, _ uint, _, _ int) ([]dto.ServerWithOntime, int64, error) {
					return nil, 0, errors.New("db error")
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
