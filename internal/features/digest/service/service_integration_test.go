package service

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
	authrepo "github.com/minhnbnt/uptime-monitor/internal/features/auth/repository"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/features/ontime/repository"
	ontimesvc "github.com/minhnbnt/uptime-monitor/internal/features/ontime/service"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

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

		if err := config.RunMigration(db); err != nil {
			fmt.Fprintf(os.Stderr, "run migration: %v\n", err)
			os.Exit(1)
		}

		testDB = db
		testDB.Create(&domain.User{
			Model:    gorm.Model{ID: 1},
			Email:    "test@test.com",
			Username: "test",
			Password: "x",
			Name:     "Test",
		})
		testDB.Create(&domain.User{
			Model:    gorm.Model{ID: 2},
			Email:    "other@test.com",
			Username: "other",
			Password: "x",
			Name:     "Other",
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

func newDigestIntegrationService(tb testing.TB, mailer MailSender) *DigestService {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}

	log := logger.NewMockLogger()
	serverRepo := serverrepo.NewServerRepository(testDB)
	batcher := ontimesvc.NewBatcher(
		ontimerepo.NewOntineRepository(testDB),
		nil,
		log,
	)
	ontimeSvc := ontimesvc.NewOntimeService(serverRepo, batcher, log)

	return &DigestService{
		userRepo:   authrepo.NewUserRepository(testDB),
		serverRepo: serverRepo,
		ontimeSvc:  ontimeSvc,
		configRepo: nil,
		mailer:     mailer,
		logger:     log,
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

func seedServer(tb testing.TB, id uint, name string, createdByID uint) {
	tb.Helper()
	testDB.Create(&domain.Server{
		Model:       gorm.Model{ID: id},
		Name:        name,
		CreatedByID: createdByID,
	})
}

func seedEndpoint(tb testing.TB, id, serverID uint, url string) {
	tb.Helper()
	testDB.Create(&domain.Endpoint{
		Model:    gorm.Model{ID: id},
		ServerID: serverID,
		URL:      url,
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

func readExcelRows(tb testing.TB, r io.Reader) [][]string {
	tb.Helper()
	data, err := io.ReadAll(r)
	if err != nil {
		tb.Fatalf("read attachment: %v", err)
	}
	xl, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		tb.Fatalf("open excel: %v", err)
	}
	defer xl.Close()

	rows, err := xl.GetRows("Sheet1")
	if err != nil {
		tb.Fatalf("get rows: %v", err)
	}
	return rows
}

func TestIntegration_SendReport_WithServers(t *testing.T) {
	truncateTables(t)

	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	seedServer(t, 1, "server-a", 1)
	seedEndpoint(t, 1, 1, "https://example.com/a")
	seedEvent(t, 1, domain.StatusOn, now.Add(-24*time.Hour))

	var capturedAttachment io.Reader
	mailer := &mockMailer{
		sendFn: func(_ string, _ string, attachment io.Reader) error {
			capturedAttachment = attachment
			return nil
		},
	}

	svc := newDigestIntegrationService(t, mailer)
	from := now.Add(-7 * 24 * time.Hour)
	if err := svc.SendReport(t.Context(), 1, from); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedAttachment == nil {
		t.Fatal("mailer.Send was not called")
	}

	rows := readExcelRows(t, capturedAttachment)
	// Header + 1 server row
	if len(rows) != 2 {
		t.Fatalf("got %d rows (incl header), want 2", len(rows))
	}
	if rows[0][0] != "Server Name" {
		t.Errorf("header[0] = %q, want Server Name", rows[0][0])
	}
	if rows[1][0] != "server-a" {
		t.Errorf("row[1] server = %q, want server-a", rows[1][0])
	}
}

func TestIntegration_SendReport_RespectsUserScoping(t *testing.T) {
	truncateTables(t)

	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

	// User 1's data
	seedServer(t, 1, "user1-server", 1)
	seedEndpoint(t, 1, 1, "https://u1.com")
	seedEvent(t, 1, domain.StatusOn, now.Add(-24*time.Hour))

	// User 2's data
	seedServer(t, 2, "user2-server", 2)
	seedEndpoint(t, 2, 2, "https://u2.com")
	seedEvent(t, 2, domain.StatusOff, now.Add(-24*time.Hour))

	var capturedAttachment io.Reader
	mailer := &mockMailer{
		sendFn: func(_ string, _ string, attachment io.Reader) error {
			capturedAttachment = attachment
			return nil
		},
	}

	svc := newDigestIntegrationService(t, mailer)
	from := now.Add(-7 * 24 * time.Hour)
	if err := svc.SendReport(t.Context(), 1, from); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows := readExcelRows(t, capturedAttachment)
	// Only user1's server
	if len(rows) != 2 {
		t.Fatalf("got %d rows (incl header), want 2", len(rows))
	}
	if rows[1][0] != "user1-server" {
		t.Errorf("row[1] server = %q, want user1-server", rows[1][0])
	}
}

func TestIntegration_SendReport_ClampsDateRange(t *testing.T) {
	truncateTables(t)

	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

	seedServer(t, 1, "server-a", 1)
	seedEndpoint(t, 1, 1, "https://example.com/a")
	// Event 40 days ago — should be excluded by clamp (30 day limit)
	seedEvent(t, 1, domain.StatusOn, now.Add(-40*24*time.Hour))
	// Event 10 days ago — should be included
	seedEvent(t, 1, domain.StatusOff, now.Add(-10*24*time.Hour))

	var capturedAttachment io.Reader
	mailer := &mockMailer{
		sendFn: func(_ string, _ string, attachment io.Reader) error {
			capturedAttachment = attachment
			return nil
		},
	}

	svc := newDigestIntegrationService(t, mailer)
	from := now.Add(-40 * 24 * time.Hour)
	if err := svc.SendReport(t.Context(), 1, from); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows := readExcelRows(t, capturedAttachment)
	// Header + 1 server (only events within 30 day clamp)
	if len(rows) != 2 {
		t.Fatalf("got %d rows (incl header), want 2", len(rows))
	}
	if rows[1][0] != "server-a" {
		t.Errorf("row[1] server = %q, want server-a", rows[1][0])
	}
}

func TestIntegration_SendReport_NoEvents(t *testing.T) {
	truncateTables(t)

	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	seedServer(t, 1, "server-a", 1)
	seedEndpoint(t, 1, 1, "https://example.com/a")

	var capturedAttachment io.Reader
	mailer := &mockMailer{
		sendFn: func(_ string, _ string, attachment io.Reader) error {
			capturedAttachment = attachment
			return nil
		},
	}

	svc := newDigestIntegrationService(t, mailer)
	if err := svc.SendReport(t.Context(), 1, now.Add(-7*24*time.Hour)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedAttachment == nil {
		t.Fatal("mailer.Send was not called")
	}

	rows := readExcelRows(t, capturedAttachment)
	// Header + 1 server row (server exists, stats show 0%)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
}

func TestIntegration_SendReport_MailerNotCalledWhenUserNotFound(t *testing.T) {
	truncateTables(t)

	mailer := &mockMailer{
		sendFn: func(_ string, _ string, _ io.Reader) error {
			t.Error("mailer.Send should not be called")
			return nil
		},
	}

	svc := newDigestIntegrationService(t, mailer)
	err := svc.SendReport(t.Context(), 999, time.Now())
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}
