package repository

import (
	"testing"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

func TestServerRepository_CountByStatus(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testcontainers.SkipIfShort(t)
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	s1 := &domain.Server{Name: "s1", CreatedByID: 1}
	if err := repo.Create(t.Context(), s1); err != nil {
		t.Fatalf("Create s1: %v", err)
	}
	s2 := &domain.Server{Name: "s2", CreatedByID: 1}
	if err := repo.Create(t.Context(), s2); err != nil {
		t.Fatalf("Create s2: %v", err)
	}
	s3 := &domain.Server{Name: "s3", CreatedByID: 2}
	if err := repo.Create(t.Context(), s3); err != nil {
		t.Fatalf("Create s3: %v", err)
	}

	testDB.Create(&domain.Endpoint{ServerID: s1.ID, URL: "https://a.com", Method: "GET", MonitorStatus: domain.StatusOn})
	testDB.Create(&domain.Endpoint{ServerID: s2.ID, URL: "https://b.com", Method: "POST", MonitorStatus: domain.StatusOff})

	t.Run("counts by status", func(t *testing.T) {
		total, online, offline, err := repo.CountByStatus(t.Context(), 1)
		if err != nil {
			t.Fatalf("CountByStatus: %v", err)
		}
		if total != 2 {
			t.Errorf("total = %d, want 2", total)
		}
		if online != 1 {
			t.Errorf("online = %d, want 1", online)
		}
		if offline != 1 {
			t.Errorf("offline = %d, want 1", offline)
		}
	})

	t.Run("different user returns zero", func(t *testing.T) {
		total, online, offline, err := repo.CountByStatus(t.Context(), 99)
		if err != nil {
			t.Fatalf("CountByStatus: %v", err)
		}
		if total != 0 {
			t.Errorf("total = %d, want 0", total)
		}
		if online != 0 {
			t.Errorf("online = %d, want 0", online)
		}
		if offline != 0 {
			t.Errorf("offline = %d, want 0", offline)
		}
	})

	t.Run("no endpoints excluded from count", func(t *testing.T) {
		s4 := &domain.Server{Name: "s4", CreatedByID: 1}
		if err := repo.Create(t.Context(), s4); err != nil {
			t.Fatalf("Create s4: %v", err)
		}

		total, _, _, err := repo.CountByStatus(t.Context(), 1)
		if err != nil {
			t.Fatalf("CountByStatus: %v", err)
		}
		if total != 2 {
			t.Errorf("total = %d, want 2 (server without endpoint excluded)", total)
		}
	})
}

func TestServerRepository_CreateAndGetByID(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	s := &domain.Server{Name: "test-server", CreatedByID: 1}
	err := repo.Create(t.Context(), s)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.ID == 0 {
		t.Fatal("ID not backfilled")
	}

	got, err := repo.GetByID(t.Context(), s.ID)
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
	testcontainers.SkipIfShort(t)
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	_, err := repo.GetByID(t.Context(), 999)
	if err == nil {
		t.Fatal("expected error for non-existent server")
	}
}

func TestServerRepository_Update(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	s := &domain.Server{Name: "original", CreatedByID: 1}
	if err := repo.Create(t.Context(), s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	s.Name = "updated"
	err := repo.Update(t.Context(), s)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(t.Context(), s.ID)
	if got.Name != "updated" {
		t.Errorf("Name = %q, want %q", got.Name, "updated")
	}
}

func TestServerRepository_Update_NotFound(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	s := &domain.Server{Name: "nope", CreatedByID: 1}
	s.ID = 999
	err := repo.Update(t.Context(), s)
	if err == nil {
		t.Fatal("expected error for non-existent server")
	}
}

func TestServerRepository_Delete(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	s := &domain.Server{Name: "delete-me", CreatedByID: 1}
	if err := repo.Create(t.Context(), s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	err := repo.Delete(t.Context(), s.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = repo.GetByID(t.Context(), s.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestServerRepository_Delete_NotFound(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	err := repo.Delete(t.Context(), 999)
	if err == nil {
		t.Fatal("expected error for non-existent server")
	}
}

func TestServerRepository_List(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	for i := range 3 {
		s := &domain.Server{Name: "srv", CreatedByID: 1}
		if err := repo.Create(t.Context(), s); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}

	t.Run("all", func(t *testing.T) {
		servers, err := repo.List(t.Context(), 1, 10, 0)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(servers) != 3 {
			t.Errorf("got %d servers, want 3", len(servers))
		}
	})

	t.Run("limit", func(t *testing.T) {
		servers, err := repo.List(t.Context(), 1, 2, 0)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(servers) != 2 {
			t.Errorf("got %d servers, want 2", len(servers))
		}
	})

	t.Run("offset", func(t *testing.T) {
		servers, err := repo.List(t.Context(), 1, 10, 2)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(servers) != 1 {
			t.Errorf("got %d servers, want 1", len(servers))
		}
	})

	t.Run("different user", func(t *testing.T) {
		servers, err := repo.List(t.Context(), 2, 10, 0)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(servers) != 0 {
			t.Errorf("got %d servers, want 0", len(servers))
		}
	})
}

func TestServerRepository_Count(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	for i := range 3 {
		s := &domain.Server{Name: "srv", CreatedByID: 1}
		if err := repo.Create(t.Context(), s); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}

	count, err := repo.Count(t.Context(), 1)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 3 {
		t.Errorf("got %d, want 3", count)
	}
}

func TestServerRepository_BatchCreateServers(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)
	repo := &ServerRepository{db: testDB}

	servers := []domain.Server{
		{Name: "batch-a", CreatedByID: 1},
		{Name: "batch-b", CreatedByID: 1},
	}
	err := repo.BatchCreateServers(t.Context(), servers)
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
		got, err := repo.GetByID(t.Context(), s.ID)
		if err != nil {
			t.Fatalf("GetByID(%d): %v", s.ID, err)
		}
		if got.Name != s.Name {
			t.Errorf("Name = %q, want %q", got.Name, s.Name)
		}
	}
}
