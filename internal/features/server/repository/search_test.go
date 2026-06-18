package repository

import (
	"testing"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

func searcher() *ParadeDBSearcher {
	return &ParadeDBSearcher{db: testDB}
}

func seedServers(tb testing.TB, servers []domain.Server) {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	for _, s := range servers {
		if err := testDB.Create(&s).Error; err != nil {
			tb.Fatalf("seed server: %v", err)
		}
	}
}

func truncateServers(tb testing.TB) {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	testDB.Exec("TRUNCATE TABLE servers CASCADE")
}

func TestSearchIntegration_Search(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	truncateServers(t)
	seedServers(t, []domain.Server{
		{Model: gorm.Model{ID: 1}, Name: "nginx web server", Status: domain.StatusActive, CreatedByID: 1},
		{Model: gorm.Model{ID: 2}, Name: "postgres database", Status: domain.StatusActive, CreatedByID: 1},
		{Model: gorm.Model{ID: 3}, Name: "redis cache", Status: domain.StatusPaused, CreatedByID: 1},
		{Model: gorm.Model{ID: 4}, Name: "web application frontend", Status: domain.StatusActive, CreatedByID: 1},
		{Model: gorm.Model{ID: 5}, Name: "api gateway", Status: domain.StatusPaused, CreatedByID: 1},
	})

	s := searcher()

	t.Run("search by keyword", func(t *testing.T) {
		results, total, err := s.Search(t.Context(), dto.SearchParams{Q: "web"}, 1)
		if err != nil {
			t.Fatalf("Search error: %v", err)
		}
		if total != 2 {
			t.Errorf("total = %d, want 2", total)
		}
		if len(results) != 2 {
			t.Fatalf("got %d results, want 2", len(results))
		}
	})

	t.Run("search no results", func(t *testing.T) {
		results, total, err := s.Search(t.Context(), dto.SearchParams{Q: "nonexistent"}, 1)
		if err != nil {
			t.Fatalf("Search error: %v", err)
		}
		if total != 0 {
			t.Errorf("total = %d, want 0", total)
		}
		if len(results) != 0 {
			t.Errorf("got %d results, want 0", len(results))
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		status := domain.StatusPaused
		results, total, err := s.Search(t.Context(), dto.SearchParams{Status: &status}, 1)
		if err != nil {
			t.Fatalf("Search error: %v", err)
		}
		if total != 2 {
			t.Errorf("total = %d, want 2", total)
		}
		for _, r := range results {
			if r.Status != domain.StatusPaused {
				t.Errorf("expected paused, got %q", r.Status)
			}
		}
	})

	t.Run("keyword with status filter", func(t *testing.T) {
		status := domain.StatusPaused
		results, total, err := s.Search(t.Context(), dto.SearchParams{Q: "api", Status: &status}, 1)
		if err != nil {
			t.Fatalf("Search error: %v", err)
		}
		if total != 1 {
			t.Errorf("total = %d, want 1", total)
		}
		if len(results) != 1 || results[0].ID != 5 {
			t.Errorf("expected api gateway (ID=5), got %+v", results)
		}
	})

	t.Run("different user sees nothing", func(t *testing.T) {
		results, total, err := s.Search(t.Context(), dto.SearchParams{Q: "web"}, 99)
		if err != nil {
			t.Fatalf("Search error: %v", err)
		}
		if total != 0 {
			t.Errorf("total = %d, want 0", total)
		}
		if len(results) != 0 {
			t.Errorf("got %d results, want 0", len(results))
		}
	})

	t.Run("pagination", func(t *testing.T) {
		results, total, err := s.Search(t.Context(), dto.SearchParams{From: 0, To: 2}, 1)
		if err != nil {
			t.Fatalf("Search error: %v", err)
		}
		if total != 5 {
			t.Errorf("total = %d, want 5", total)
		}
		if len(results) != 2 {
			t.Errorf("got %d results, want 2", len(results))
		}
	})

	t.Run("sort by name ascending", func(t *testing.T) {
		results, _, err := s.Search(t.Context(), dto.SearchParams{SortBy: "name", SortOrder: "asc"}, 1)
		if err != nil {
			t.Fatalf("Search error: %v", err)
		}
		if len(results) < 2 {
			t.Fatalf("need at least 2 results, got %d", len(results))
		}
		if results[0].Name > results[1].Name {
			t.Errorf("not sorted ascending: %q > %q", results[0].Name, results[1].Name)
		}
	})
}
