package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

func dtoServer(id uint, name string, t time.Time) dto.Server {
	return dto.Server{ID: id, Name: name, Status: "active", CreatedAt: t, UpdatedAt: t}
}

func TestServerHandler_ListServers(t *testing.T) {
	now := time.Now()
	srv := utils.NewPageValidator(30)

	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			service: &mockServerService{
				listServersFn: func(_ context.Context, page, perPage int) ([]dto.Server, error) {
					if page != 2 || perPage != 10 {
						t.Errorf("ListServers(%d, %d)", page, perPage)
					}
					return []dto.Server{dtoServer(1, "s1", now)}, nil
				},
			},
			pageValidator: srv,
		}
		c, w := newGinContext("GET", "/api/v1/servers?page=2&per_page=10", "")
		h.ListServers(c, api.ListServersParams{Page: intPtr(2), PerPage: intPtr(10)})

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var resp api.ServerListResponse
		parseJSON(w, &resp)
		if len(resp.Data) != 1 || resp.Data[0].Name != "s1" {
			t.Errorf("unexpected data: %+v", resp.Data)
		}
	})

	t.Run("invalid page", func(t *testing.T) {
		h := &ServerHandler{pageValidator: srv}
		c, w := newGinContext("GET", "/api/v1/servers?page=0", "")
		h.ListServers(c, api.ListServersParams{Page: intPtr(0)})

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &ServerHandler{
			service: &mockServerService{
				listServersFn: func(_ context.Context, _, _ int) ([]dto.Server, error) {
					return nil, errors.New("db error")
				},
			},
			pageValidator: srv,
		}
		c, w := newGinContext("GET", "/api/v1/servers", "")
		h.ListServers(c, api.ListServersParams{})

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestServerHandler_CreateServer(t *testing.T) {
	now := time.Now()
	val := &RequestValidator{v: validator.New()}

	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			service: &mockServerService{
				createServerFn: func(_ context.Context, req dto.CreateServerRequest) (*dto.Server, error) {
					s := dtoServer(1, req.Name, now)
					return &s, nil
				},
			},
			validator: val,
		}
		c, w := newGinContext("POST", "/api/v1/servers", `{"name":"new-srv"}`)
		h.CreateServer(c)

		if w.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
		}
		var resp api.ServerResponse
		parseJSON(w, &resp)
		if resp.Data.Name != "new-srv" {
			t.Errorf("name = %q", resp.Data.Name)
		}
	})

	t.Run("bad json", func(t *testing.T) {
		h := &ServerHandler{validator: val}
		c, w := newGinContext("POST", "/api/v1/servers", `{bad`)
		h.CreateServer(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &ServerHandler{
			service: &mockServerService{
				createServerFn: func(_ context.Context, _ dto.CreateServerRequest) (*dto.Server, error) {
					return nil, errors.New("db error")
				},
			},
			validator: val,
		}
		c, w := newGinContext("POST", "/api/v1/servers", `{"name":"x"}`)
		h.CreateServer(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestServerHandler_GetServer(t *testing.T) {
	now := time.Now()

	t.Run("found", func(t *testing.T) {
		h := &ServerHandler{
			service: &mockServerService{
				getServerFn: func(_ context.Context, id uint) (*dto.Server, error) {
					s := dtoServer(id, "found", now)
					return &s, nil
				},
			},
		}
		c, w := newGinContext("GET", "/api/v1/servers/5", "")
		h.GetServer(c, 5)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var resp api.ServerResponse
		parseJSON(w, &resp)
		if resp.Data.Id != 5 || resp.Data.Name != "found" {
			t.Errorf("unexpected: %+v", resp.Data)
		}
	})

	t.Run("not found", func(t *testing.T) {
		h := &ServerHandler{
			service: &mockServerService{
				getServerFn: func(_ context.Context, _ uint) (*dto.Server, error) {
					return nil, errors.New("not found")
				},
			},
		}
		c, w := newGinContext("GET", "/api/v1/servers/99", "")
		h.GetServer(c, 99)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestServerHandler_UpdateServer(t *testing.T) {
	now := time.Now()
	val := &RequestValidator{v: validator.New()}

	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			service: &mockServerService{
				updateServerFn: func(_ context.Context, id uint, req dto.UpdateServerRequest) (*dto.Server, error) {
					s := dtoServer(id, *req.Name, now)
					return &s, nil
				},
			},
			validator: val,
		}
		c, w := newGinContext("PUT", "/api/v1/servers/3", `{"name":"updated"}`)
		h.UpdateServer(c, 3)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var resp api.ServerResponse
		parseJSON(w, &resp)
		if resp.Data.Name != "updated" {
			t.Errorf("name = %q", resp.Data.Name)
		}
	})

	t.Run("bad json", func(t *testing.T) {
		h := &ServerHandler{validator: val}
		c, w := newGinContext("PUT", "/api/v1/servers/3", `{bad`)
		h.UpdateServer(c, 3)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		h := &ServerHandler{
			service: &mockServerService{
				updateServerFn: func(_ context.Context, _ uint, _ dto.UpdateServerRequest) (*dto.Server, error) {
					return nil, errors.New("not found")
				},
			},
			validator: val,
		}
		c, w := newGinContext("PUT", "/api/v1/servers/99", `{"name":"x"}`)
		h.UpdateServer(c, 99)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestServerHandler_DeleteServer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			service: &mockServerService{
				deleteServerFn: func(_ context.Context, id uint) error {
					return nil
				},
			},
		}
		c, w := newGinContext("DELETE", "/api/v1/servers/4", "")
		h.DeleteServer(c, 4)
		c.Writer.WriteHeaderNow()

		if w.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
		}
	})

	t.Run("not found", func(t *testing.T) {
		h := &ServerHandler{
			service: &mockServerService{
				deleteServerFn: func(_ context.Context, _ uint) error {
					return errors.New("not found")
				},
			},
		}
		c, w := newGinContext("DELETE", "/api/v1/servers/99", "")
		h.DeleteServer(c, 99)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestServerHandler_ListServersOntime(t *testing.T) {
	now := time.Now()
	srv := utils.NewPageValidator(30)

	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			ontimeService: &mockOntimeService{
				listServersWithOntimeFn: func(_ context.Context, page, perPage int) ([]dto.ServerWithOntime, int64, error) {
					return []dto.ServerWithOntime{
						{Server: dtoServer(1, "s1", now), OntimeStats: []dto.OntimeStats{{Date: now, Stats: 95.5}}},
					}, 1, nil
				},
			},
			pageValidator: srv,
		}
		c, w := newGinContext("GET", "/api/v1/servers/ontime?page=1&per_page=20", "")
		h.ListServersOntime(c, api.ListServersOntimeParams{Page: intPtr(1), PerPage: intPtr(20)})

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var resp api.ServerOntimeListResponse
		parseJSON(w, &resp)
		if len(resp.Data) != 1 || resp.Data[0].Server.Name != "s1" {
			t.Errorf("unexpected data: %+v", resp.Data)
		}
		if resp.Meta.Total == nil || *resp.Meta.Total != 1 {
			t.Errorf("total = %v", resp.Meta.Total)
		}
	})

	t.Run("invalid page", func(t *testing.T) {
		h := &ServerHandler{pageValidator: srv}
		c, w := newGinContext("GET", "/api/v1/servers/ontime?page=0", "")
		h.ListServersOntime(c, api.ListServersOntimeParams{Page: intPtr(0)})

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &ServerHandler{
			ontimeService: &mockOntimeService{
				listServersWithOntimeFn: func(_ context.Context, _, _ int) ([]dto.ServerWithOntime, int64, error) {
					return nil, 0, errors.New("db error")
				},
			},
			pageValidator: srv,
		}
		c, w := newGinContext("GET", "/api/v1/servers/ontime", "")
		h.ListServersOntime(c, api.ListServersOntimeParams{})

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}

func intPtr(i int) *int {
	return &i
}
