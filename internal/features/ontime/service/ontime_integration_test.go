package ontime

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/features/ontime/repository"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

// ---------- container lifecycle ----------

var testDB *gorm.DB

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()

		container, dsn := startPostgres(ctx)
		defer func() { _ = container.Terminate(ctx) }()

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "gorm open: %v\n", err)
			os.Exit(1)
		}

		schemas := []any{
			&domain.User{},
			&domain.Server{},
			&domain.Endpoint{},
			&domain.ServerEvent{},
		}
		if err := db.AutoMigrate(schemas...); err != nil {
			fmt.Fprintf(os.Stderr, "auto-migrate: %v\n", err)
			os.Exit(1)
		}

		testDB = db
		// Seed a default user so servers can reference it via CreatedByID.
		testDB.Create(&domain.User{
			Model:    gorm.Model{ID: 1},
			Email:    "test@test.com",
			Username: "test",
			Password: "x",
			Name:     "Test",
		})
	}
	os.Exit(m.Run())
}

func startPostgres(ctx context.Context) (testcontainers.Container, string) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "uptime_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60 * time.Second),
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

// ---------- helpers ----------

func newService(tb testing.TB) *OntimeService {
	tb.Helper()

	if testing.Short() {
		tb.Skip("skipping integration test")
		return nil
	}

	return &OntimeService{
		serverRepository: serverrepo.NewServerRepository(testDB),
		batcher: &Batcher{
			ontineRepository: ontimerepo.NewOntineRepository(testDB),
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, _ []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error) {
					return make(map[dto.BatchGetOntimeItem]float64), nil
				},
				mSetFn: func(_ context.Context, _ map[dto.BatchGetOntimeItem]float64) error {
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		},
	}
}

func truncateTables(tb testing.TB) {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	for _, tbl := range []string{"server_events", "endpoints", "servers"} {
		testDB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", tbl))
	}
}

func seedServer(tb testing.TB, id uint, name string, createdAt time.Time) {
	tb.Helper()
	testDB.Create(&domain.Server{
		Model:       gorm.Model{ID: id, CreatedAt: createdAt, UpdatedAt: createdAt},
		Name:        name,
		CreatedByID: 1,
	})
}

func seedEndpoint(tb testing.TB, id, serverID uint) {
	tb.Helper()
	testDB.Create(&domain.Endpoint{
		Model:    gorm.Model{ID: id},
		ServerID: serverID,
		URL:      "https://example.com",
		Method:   "GET",
	})
}

func seedEvent(tb testing.TB, endpointID uint, status domain.ServerStatus, tm time.Time) {
	tb.Helper()
	testDB.Create(&domain.ServerEvent{
		ID:         uuid.New(),
		EndpointID: endpointID,
		Status:     status,
		Time:       tm,
	})
}

// ---------- BatchGetOntime ----------

func TestIntegration_BatchGetOntime_CacheMiss(t *testing.T) {
	truncateTables(t)

	now := oDay(2026, 6, 1)
	seedServer(t, 1, "s1", now.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)
	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 6, 0))
	seedEvent(t, 1, domain.StatusOff, oTm(2026, 6, 1, 18, 0))

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if len(results[0].Result) != 1 {
		t.Fatalf("len(Result) = %d, want 1", len(results[0].Result))
	}
	if results[0].Result[0].Stats <= 0 {
		t.Errorf("Stats = %f, want > 0", results[0].Result[0].Stats)
	}
}

func TestIntegration_BatchGetOntime_AllOn(t *testing.T) {
	truncateTables(t)

	now := oDay(2026, 6, 1)
	seedServer(t, 1, "s1", now.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)
	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 0, 0))

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Result[0].Stats != 100 {
		t.Errorf("Stats = %f, want 100", results[0].Result[0].Stats)
	}
}

func TestIntegration_BatchGetOntime_AllOff(t *testing.T) {
	truncateTables(t)

	now := oDay(2026, 6, 1)
	seedServer(t, 1, "s1", now.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)
	seedEvent(t, 1, domain.StatusOff, oTm(2026, 6, 1, 0, 0))

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Result[0].Stats != 0 {
		t.Errorf("Stats = %f, want 0", results[0].Result[0].Stats)
	}
}

func TestIntegration_BatchGetOntime_NoEvents(t *testing.T) {
	truncateTables(t)

	now := oDay(2026, 6, 1)
	seedServer(t, 1, "s1", now.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Result[0].Stats != 0 {
		t.Errorf("Stats = %f, want 0 (no events)", results[0].Result[0].Stats)
	}
}

func TestIntegration_BatchGetOntime_EmptyRequest(t *testing.T) {
	truncateTables(t)

	svc := newService(t)
	results, err := svc.batcher.BatchGetOntime(t.Context(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}

// ---------- ListServersWithOntime ----------

func TestIntegration_ListServersWithOntime(t *testing.T) {
	truncateTables(t)

	createdAt := oDay(2026, 6, 1).Add(-48 * time.Hour)
	seedServer(t, 1, "s1", createdAt)
	seedServer(t, 2, "s2", createdAt)
	seedEndpoint(t, 1, 1)
	seedEndpoint(t, 2, 2)
	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 6, 0))
	seedEvent(t, 2, domain.StatusOff, oTm(2026, 6, 2, 0, 0))

	svc := newService(t)
	results, total, err := svc.ListServersWithOntime(t.Context(), 1, 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	// Each server should have ontime stats for its date range
	for _, r := range results {
		if len(r.OntimeStats) == 0 {
			t.Errorf("server %s: ontime_stats is empty", r.Server.Name)
		}
		for _, stat := range r.OntimeStats {
			if stat.Stats < 0 || stat.Stats > 100 {
				t.Errorf("server %s, date %v: Stats = %f, out of range [0,100]",
					r.Server.Name, stat.Date, stat.Stats)
			}
		}
	}
}

// ---------- GetServerWithOntime ----------

func TestIntegration_GetServerWithOntime(t *testing.T) {
	truncateTables(t)

	createdAt := oDay(2026, 6, 1).Add(-48 * time.Hour)
	seedServer(t, 1, "s1", createdAt)
	seedEndpoint(t, 1, 1)
	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 6, 0))

	svc := newService(t)
	result, err := svc.GetServerWithOntime(t.Context(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Server.Name != "s1" {
		t.Errorf("Server.Name = %q, want s1", result.Server.Name)
	}
	if len(result.OntimeStats) == 0 {
		t.Error("ontime_stats is empty")
	}
}

func TestIntegration_GetServerWithOntime_NotFound(t *testing.T) {
	truncateTables(t)

	svc := newService(t)
	_, err := svc.GetServerWithOntime(t.Context(), 999)
	if err == nil {
		t.Fatal("expected error for non-existent server")
	}
}

// ---------- URL encoding regression ----------

func TestIntegration_BatchGetOntime_URLSpecialChars(t *testing.T) {
	truncateTables(t)

	now := oDay(2026, 6, 1)
	seedServer(t, 1, "s1", now.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)

	// Server with URL containing special characters
	seedServer(t, 2, "s2", now.Add(-48*time.Hour))
	testDB.Create(&domain.Endpoint{
		Model:    gorm.Model{ID: 2},
		ServerID: 2,
		URL:      "https://example.com/path?q=1&r=2",
		Method:   "GET",
	})

	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 6, 0))

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{
		{ServerID: 1, Date: now},
		{ServerID: 2, Date: now},
	}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2", len(results))
	}
}

// ---------- today vs past day ----------

func TestIntegration_BatchGetOntime_Today(t *testing.T) {
	truncateTables(t)

	today := oDay(time.Now().Year(), int(time.Now().Month()), time.Now().Day())
	onTime := today.Add(6 * time.Hour)
	if time.Now().Before(onTime) {
		t.Skip("event at 06:00 UTC hasn't happened yet — skip")
	}

	seedServer(t, 1, "s1", today.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)
	seedEvent(t, 1, domain.StatusOn, onTime)

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: today}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Result) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}

	// Today should have partial coverage (from 06:00 to now)
	got := results[0].Result[0].Stats
	if got <= 0 {
		t.Errorf("Stats = %f, want > 0 for today", got)
	}
}
