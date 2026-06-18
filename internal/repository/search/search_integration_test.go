package search

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()

		container, dsn := startParadeDB(ctx)
		defer func() { _ = container.Terminate(ctx) }()

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "gorm open: %v\n", err)
			os.Exit(1)
		}

		if err := db.AutoMigrate(&domain.Server{}); err != nil {
			fmt.Fprintf(os.Stderr, "auto-migrate: %v\n", err)
			os.Exit(1)
		}

		if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_search").Error; err != nil {
			fmt.Fprintf(os.Stderr, "create extension: %v\n", err)
			os.Exit(1)
		}

		if err := db.Exec(`CREATE INDEX IF NOT EXISTS servers_search_idx ON servers USING bm25 (id, name) WITH (key_field='id')`).Error; err != nil {
			fmt.Fprintf(os.Stderr, "create bm25 index: %v\n", err)
			os.Exit(1)
		}

		testDB = db
	}

	os.Exit(m.Run())
}

func startParadeDB(ctx context.Context) (testcontainers.Container, string) {
	req := testcontainers.ContainerRequest{
		Image:        "paradedb/paradedb:pg17",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "uptime_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(120 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start container: %v\n", err)
		os.Exit(1)
	}

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "container host: %v\n", err)
		os.Exit(1)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		fmt.Fprintf(os.Stderr, "container port: %v\n", err)
		os.Exit(1)
	}

	dsn := fmt.Sprintf(
		"postgres://test:test@%s:%s/uptime_test?sslmode=disable",
		host, port.Port(),
	)

	return container, dsn
}

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
