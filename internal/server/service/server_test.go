package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

func TestServerService_ListServers(t *testing.T) {
	now := time.Now()
	domainServers := []domain.Server{
		{
			Model:    gormModel(1, now),
			Name:     "server-a",
			Status:   domain.StatusActive,
			Endpoint: nil,
		},
		{
			Model:    gormModel(2, now),
			Name:     "server-b",
			Status:   domain.StatusPaused,
			Endpoint: nil,
		},
	}

	t.Run("success", func(t *testing.T) {
		svc := &ServerService{serverRepository: &mockServerRepo{
			listFn: func(_ context.Context, limit, offset int) ([]domain.Server, error) {
				if limit != 10 || offset != 0 {
					t.Errorf("List(%d, %d)", limit, offset)
				}
				return domainServers, nil
			},
		}}

		got, err := svc.ListServers(t.Context(), 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d, want 2", len(got))
		}
		if got[0].Name != "server-a" || got[1].Name != "server-b" {
			t.Errorf("names: %v", got)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		svc := &ServerService{serverRepository: &mockServerRepo{
			listFn: func(_ context.Context, _, _ int) ([]domain.Server, error) {
				return nil, errors.New("db error")
			},
		}}

		_, err := svc.ListServers(t.Context(), 1, 10)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestServerService_CreateServer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var saved *domain.Server
		svc := &ServerService{serverRepository: &mockServerRepo{
			createFn: func(_ context.Context, s *domain.Server) error {
				s.ID = 42
				s.CreatedAt = time.Now()
				s.UpdatedAt = time.Now()
				saved = s
				return nil
			},
		}}

		req := dto.CreateServerRequest{Name: "my-server"}
		got, err := svc.CreateServer(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 42 {
			t.Errorf("ID = %d, want 42", got.ID)
		}
		if got.Name != "my-server" {
			t.Errorf("Name = %q, want my-server", got.Name)
		}
		if got.Status != domain.StatusActive {
			t.Errorf("Status = %q, want active", got.Status)
		}
		if saved.Status != domain.StatusActive {
			t.Errorf("saved.Status = %q, want active", saved.Status)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		svc := &ServerService{serverRepository: &mockServerRepo{
			createFn: func(_ context.Context, _ *domain.Server) error {
				return errors.New("db error")
			},
		}}

		_, err := svc.CreateServer(t.Context(), dto.CreateServerRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestServerService_GetServer(t *testing.T) {
	now := time.Now()

	t.Run("found", func(t *testing.T) {
		svc := &ServerService{serverRepository: &mockServerRepo{
			getByIDFn: func(_ context.Context, id uint) (*domain.Server, error) {
				return &domain.Server{
					Model:  gormModel(id, now),
					Name:   "found-server",
					Status: domain.StatusActive,
				}, nil
			},
		}}

		got, err := svc.GetServer(t.Context(), 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 7 {
			t.Errorf("ID = %d, want 7", got.ID)
		}
		if got.Name != "found-server" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc := &ServerService{serverRepository: &mockServerRepo{
			getByIDFn: func(_ context.Context, _ uint) (*domain.Server, error) {
				return nil, errors.New("not found")
			},
		}}

		_, err := svc.GetServer(t.Context(), 99)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestServerService_UpdateServer(t *testing.T) {
	now := time.Now()
	existing := &domain.Server{
		Model:  gormModel(1, now),
		Name:   "original",
		Status: domain.StatusActive,
	}

	t.Run("update name and status", func(t *testing.T) {
		var updated *domain.Server
		svc := &ServerService{serverRepository: &mockServerRepo{
			getByIDFn: func(_ context.Context, _ uint) (*domain.Server, error) {
				cp := *existing
				return &cp, nil
			},
			updateFn: func(_ context.Context, s *domain.Server) error {
				updated = s
				return nil
			},
		}}

		name := "renamed"
		status := domain.StatusPaused
		req := dto.UpdateServerRequest{Name: &name, Status: &status}
		got, err := svc.UpdateServer(t.Context(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "renamed" {
			t.Errorf("Name = %q, want renamed", got.Name)
		}
		if got.Status != domain.StatusPaused {
			t.Errorf("Status = %q, want paused", got.Status)
		}
		if updated.Name != "renamed" || updated.Status != domain.StatusPaused {
			t.Errorf("updated = %+v", updated)
		}
	})

	t.Run("nil fields leave unchanged", func(t *testing.T) {
		var updated *domain.Server
		svc := &ServerService{serverRepository: &mockServerRepo{
			getByIDFn: func(_ context.Context, _ uint) (*domain.Server, error) {
				cp := *existing
				return &cp, nil
			},
			updateFn: func(_ context.Context, s *domain.Server) error {
				updated = s
				return nil
			},
		}}

		req := dto.UpdateServerRequest{}
		got, err := svc.UpdateServer(t.Context(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "original" {
			t.Errorf("Name changed to %q", got.Name)
		}
		if updated.Name != "original" || updated.Status != domain.StatusActive {
			t.Errorf("updated should be unchanged: %+v", updated)
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc := &ServerService{serverRepository: &mockServerRepo{
			getByIDFn: func(_ context.Context, _ uint) (*domain.Server, error) {
				return nil, errors.New("not found")
			},
		}}

		_, err := svc.UpdateServer(t.Context(), 99, dto.UpdateServerRequest{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("update error", func(t *testing.T) {
		svc := &ServerService{serverRepository: &mockServerRepo{
			getByIDFn: func(_ context.Context, _ uint) (*domain.Server, error) {
				cp := *existing
				return &cp, nil
			},
			updateFn: func(_ context.Context, _ *domain.Server) error {
				return errors.New("update failed")
			},
		}}

		_, err := svc.UpdateServer(t.Context(), 1, dto.UpdateServerRequest{})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestServerService_DeleteServer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var deleted uint
		svc := &ServerService{serverRepository: &mockServerRepo{
			deleteFn: func(_ context.Context, id uint) error {
				deleted = id
				return nil
			},
		}}

		err := svc.DeleteServer(t.Context(), 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deleted != 7 {
			t.Errorf("deleted = %d, want 7", deleted)
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc := &ServerService{serverRepository: &mockServerRepo{
			deleteFn: func(_ context.Context, _ uint) error {
				return errors.New("not found")
			},
		}}

		err := svc.DeleteServer(t.Context(), 99)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
