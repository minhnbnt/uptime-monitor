package server

import (
	"context"
	"testing"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

func TestServerRepository_CreateAndGetByID(t *testing.T) {
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	s := &domain.Server{Name: "test-server", CreatedByID: 1}
	err := repo.Create(context.Background(), s)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.ID == 0 {
		t.Fatal("ID not backfilled")
	}

	got, err := repo.GetByID(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "test-server" {
		t.Errorf("Name = %q, want %q", got.Name, "test-server")
	}
	if got.CreatedByID != 1 {
		t.Errorf("CreatedByID = %d, want 1", got.CreatedByID)
	}
}

func TestServerRepository_GetByID_NotFound(t *testing.T) {
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	_, err := repo.GetByID(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for non-existent server")
	}
}

func TestServerRepository_Update(t *testing.T) {
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	s := &domain.Server{Name: "original", CreatedByID: 1}
	if err := repo.Create(context.Background(), s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	s.Name = "updated"
	err := repo.Update(context.Background(), s)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(context.Background(), s.ID)
	if got.Name != "updated" {
		t.Errorf("Name = %q, want %q", got.Name, "updated")
	}
}

func TestServerRepository_Update_NotFound(t *testing.T) {
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	s := &domain.Server{Name: "nope", CreatedByID: 1}
	s.ID = 999
	err := repo.Update(context.Background(), s)
	if err == nil {
		t.Fatal("expected error for non-existent server")
	}
}

func TestServerRepository_Delete(t *testing.T) {
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	s := &domain.Server{Name: "delete-me", CreatedByID: 1}
	if err := repo.Create(context.Background(), s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	err := repo.Delete(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = repo.GetByID(context.Background(), s.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestServerRepository_Delete_NotFound(t *testing.T) {
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	err := repo.Delete(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for non-existent server")
	}
}

func TestServerRepository_List(t *testing.T) {
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	for i := range 3 {
		s := &domain.Server{Name: "srv", CreatedByID: 1}
		if err := repo.Create(context.Background(), s); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}

	t.Run("all", func(t *testing.T) {
		servers, err := repo.List(context.Background(), 1, 10, 0)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(servers) != 3 {
			t.Errorf("got %d servers, want 3", len(servers))
		}
	})

	t.Run("limit", func(t *testing.T) {
		servers, err := repo.List(context.Background(), 1, 2, 0)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(servers) != 2 {
			t.Errorf("got %d servers, want 2", len(servers))
		}
	})

	t.Run("offset", func(t *testing.T) {
		servers, err := repo.List(context.Background(), 1, 10, 2)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(servers) != 1 {
			t.Errorf("got %d servers, want 1", len(servers))
		}
	})

	t.Run("different user", func(t *testing.T) {
		servers, err := repo.List(context.Background(), 2, 10, 0)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(servers) != 0 {
			t.Errorf("got %d servers, want 0", len(servers))
		}
	})
}

func TestServerRepository_Count(t *testing.T) {
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	for i := range 3 {
		s := &domain.Server{Name: "srv", CreatedByID: 1}
		if err := repo.Create(context.Background(), s); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}

	count, err := repo.Count(context.Background(), 1)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 3 {
		t.Errorf("got %d, want 3", count)
	}
}

func TestServerRepository_BatchCreateServers(t *testing.T) {
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	servers := []domain.Server{
		{Name: "batch-a", CreatedByID: 1},
		{Name: "batch-b", CreatedByID: 1},
	}
	err := repo.BatchCreateServers(context.Background(), servers)
	if err != nil {
		t.Fatalf("BatchCreateServers: %v", err)
	}
	if servers[0].ID == 0 {
		t.Error("servers[0].ID not backfilled")
	}
	if servers[1].ID == 0 {
		t.Error("servers[1].ID not backfilled")
	}
	if servers[0].ID == servers[1].ID {
		t.Error("servers have same ID")
	}

	for _, s := range servers {
		got, err := repo.GetByID(context.Background(), s.ID)
		if err != nil {
			t.Fatalf("GetByID(%d): %v", s.ID, err)
		}
		if got.Name != s.Name {
			t.Errorf("Name = %q, want %q", got.Name, s.Name)
		}
	}
}
