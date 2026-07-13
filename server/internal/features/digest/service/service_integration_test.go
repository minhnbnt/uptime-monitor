package service

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	digestinfra "github.com/minhnbnt/uptime-monitor/internal/features/digest/infrastructure"
	digestrepo "github.com/minhnbnt/uptime-monitor/internal/features/digest/repository"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/features/ontime/repository"
	ontimesvc "github.com/minhnbnt/uptime-monitor/internal/features/ontime/service"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

var testDB *gorm.DB
var testDSN string

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()

		container, dsn := testcontainers.StartPostgres(ctx)
		defer func() { _ = container.Terminate(ctx) }()
		testDSN = dsn
	}
	os.Exit(m.Run())
}

func initTestDB(tb testing.TB) *gorm.DB {
	tb.Helper()
	return testcontainers.CreateTestDB(tb, testDSN, func(db *gorm.DB) {
		if err := db.Create(&domain.User{
			Model:    gorm.Model{ID: 1},
			Email:    "test@test.com",
			Username: "test",
			Password: "x",
			Name:     "Test",
		}).Error; err != nil {
			tb.Fatalf("seed user 1: %v", err)
		}
		if err := db.Create(&domain.User{
			Model:    gorm.Model{ID: 2},
			Email:    "other@test.com",
			Username: "other",
			Password: "x",
			Name:     "Other",
		}).Error; err != nil {
			tb.Fatalf("seed user 2: %v", err)
		}
	})
}

func newDigestIntegrationService(tb testing.TB, mailer MailSender) *DigestService {
	tb.Helper()
	testcontainers.SkipIfShort(tb)

	log := logger.NewMockLogger()
	serverRepo := serverrepo.NewServerRepository(testDB)
	batcher := ontimesvc.NewBatcher(
		ontimerepo.NewOntineRepository(testDB),
		nil,
		log,
	)
	ontimeSvc := ontimesvc.NewOntimeService(serverRepo, batcher, log)

	return &DigestService{
		userRepo:   digestrepo.NewUserRepository(testDB),
		serverRepo: serverRepo,
		ontimeSvc:  ontimeSvc,
		configRepo: nil,
		mailer:     mailer,
		logger:     log,
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

func seedEndpoint(tb testing.TB, id, serverID uint, url string, monitorStatus ...domain.ServerStatus) {
	tb.Helper()
	e := domain.Endpoint{
		Model:    gorm.Model{ID: id},
		ServerID: serverID,
		URL:      url,
		Method:   "GET",
	}
	if len(monitorStatus) > 0 {
		e.MonitorStatus = monitorStatus[0]
	}
	testDB.Create(&e)
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

func readExcelSheet(tb testing.TB, r io.Reader, sheet string) [][]string {
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

	rows, err := xl.GetRows(sheet)
	if err != nil {
		tb.Fatalf("get rows from %s: %v", sheet, err)
	}
	return rows
}

func readExcelRows(tb testing.TB, r io.Reader) [][]string {
	return readExcelSheet(tb, r, digestinfra.ReportSheetName)
}

func TestIntegration_SendReport_WithServers(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	seedServer(t, 1, "server-a", 1)
	seedEndpoint(t, 1, 1, "https://example.com/a", domain.StatusOn)
	seedEvent(t, 1, domain.StatusOn, now.Add(-24*time.Hour))

	var capturedData []byte
	mailer := &mockMailer{
		sendFn: func(_ string, _ string, attachment io.Reader) error {
			var err error
			capturedData, err = io.ReadAll(attachment)
			return err
		},
	}

	svc := newDigestIntegrationService(t, mailer)
	from := now.Add(-7 * 24 * time.Hour)
	if err := svc.SendReport(t.Context(), 1, from); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedData == nil {
		t.Fatal("mailer.Send was not called")
	}

	rows := readExcelSheet(t, bytes.NewReader(capturedData), digestinfra.ReportSheetName)
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

	summary := readExcelSheet(t, bytes.NewReader(capturedData), digestinfra.SummarySheetName)
	if len(summary) != 4 {
		t.Fatalf("got %d summary rows, want 4", len(summary))
	}
	if summary[1][0] != "Total Servers" || summary[1][1] != "1" {
		t.Errorf("total: %v, want [Total Servers 1]", summary[1])
	}
	if summary[2][0] != "Online" || summary[2][1] != "1" {
		t.Errorf("online: %v, want [Online 1]", summary[2])
	}
	if summary[3][0] != "Offline" || summary[3][1] != "0" {
		t.Errorf("offline: %v, want [Offline 0]", summary[3])
	}
}

func TestIntegration_SendReport_RespectsUserScoping(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

	// User 1's data
	seedServer(t, 1, "user1-server", 1)
	seedEndpoint(t, 1, 1, "https://u1.com", domain.StatusOn)
	seedEvent(t, 1, domain.StatusOn, now.Add(-24*time.Hour))

	// User 2's data
	seedServer(t, 2, "user2-server", 2)
	seedEndpoint(t, 2, 2, "https://u2.com", domain.StatusOff)
	seedEvent(t, 2, domain.StatusOff, now.Add(-24*time.Hour))

	var capturedData []byte
	mailer := &mockMailer{
		sendFn: func(_ string, _ string, attachment io.Reader) error {
			var err error
			capturedData, err = io.ReadAll(attachment)
			return err
		},
	}

	svc := newDigestIntegrationService(t, mailer)
	from := now.Add(-7 * 24 * time.Hour)
	if err := svc.SendReport(t.Context(), 1, from); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows := readExcelRows(t, bytes.NewReader(capturedData))
	// Only user1's server
	if len(rows) != 2 {
		t.Fatalf("got %d rows (incl header), want 2", len(rows))
	}
	if rows[1][0] != "user1-server" {
		t.Errorf("row[1] server = %q, want user1-server", rows[1][0])
	}
}

func TestIntegration_SendReport_ClampsDateRange(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

	seedServer(t, 1, "server-a", 1)
	seedEndpoint(t, 1, 1, "https://example.com/a")
	// Event 40 days ago — should be excluded by clamp (30 day limit)
	seedEvent(t, 1, domain.StatusOn, now.Add(-40*24*time.Hour))
	// Event 10 days ago — should be included
	seedEvent(t, 1, domain.StatusOff, now.Add(-10*24*time.Hour))

	var capturedData []byte
	mailer := &mockMailer{
		sendFn: func(_ string, _ string, attachment io.Reader) error {
			var err error
			capturedData, err = io.ReadAll(attachment)
			return err
		},
	}

	svc := newDigestIntegrationService(t, mailer)
	from := now.Add(-40 * 24 * time.Hour)
	if err := svc.SendReport(t.Context(), 1, from); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows := readExcelRows(t, bytes.NewReader(capturedData))
	// Header + 1 server (only events within 30 day clamp)
	if len(rows) != 2 {
		t.Fatalf("got %d rows (incl header), want 2", len(rows))
	}
	if rows[1][0] != "server-a" {
		t.Errorf("row[1] server = %q, want server-a", rows[1][0])
	}
}

func TestIntegration_SendReport_NoEvents(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	seedServer(t, 1, "server-a", 1)
	seedEndpoint(t, 1, 1, "https://example.com/a")

	var capturedData []byte
	mailer := &mockMailer{
		sendFn: func(_ string, _ string, attachment io.Reader) error {
			var err error
			capturedData, err = io.ReadAll(attachment)
			return err
		},
	}

	svc := newDigestIntegrationService(t, mailer)
	if err := svc.SendReport(t.Context(), 1, now.Add(-7*24*time.Hour)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedData == nil {
		t.Fatal("mailer.Send was not called")
	}

	rows := readExcelRows(t, bytes.NewReader(capturedData))
	// Header + 1 server row (server exists, stats show 0%)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
}

func TestIntegration_SendReport_MailerNotCalledWhenUserNotFound(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

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
