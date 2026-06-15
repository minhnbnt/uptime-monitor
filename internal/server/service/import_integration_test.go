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

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/server/infrastructure"
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

func newImportIntegrationService(tb testing.TB) *ImportService {
	tb.Helper()

	if testing.Short() {
		tb.Skip("skipping integration test")
	}

	return &ImportService{
		serverRepository:   serverrepo.NewServerRepository(testDB),
		endpointRepository: serverrepo.NewEndpointRepository(testDB),
		excelGenerator:     &infrastructure.ExcelGenerator{},
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

func buildExcel(tb testing.TB, rows []dto.ImportRow) io.Reader {
	tb.Helper()

	xl := excelize.NewFile()
	defer xl.Close()

	headers := []string{"server_name", "url", "method", "interval_sec", "timeout_sec", "expected_code"}
	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			tb.Fatalf("cell name: %v", err)
		}
		if err := xl.SetCellValue("Sheet1", cell, h); err != nil {
			tb.Fatalf("set header: %v", err)
		}
	}

	for i, r := range rows {
		rowNum := i + 2
		vals := []any{r.Name, r.URL, r.Method, r.Interval, r.Timeout, r.ExpectedCode}
		for j, v := range vals {
			cell, err := excelize.CoordinatesToCellName(j+1, rowNum)
			if err != nil {
				tb.Fatalf("cell name: %v", err)
			}
			if err := xl.SetCellValue("Sheet1", cell, v); err != nil {
				tb.Fatalf("set value: %v", err)
			}
		}
	}

	buf := new(bytes.Buffer)
	if _, err := xl.WriteTo(buf); err != nil {
		tb.Fatalf("write excel: %v", err)
	}

	return buf
}

func TestIntegration_ImportServers_Success(t *testing.T) {
	truncateTables(t)

	rows := []dto.ImportRow{
		{Name: "server-a", URL: "https://a.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		{Name: "server-b", URL: "https://b.org/ping", Method: "POST", Interval: 60, Timeout: 15, ExpectedCode: 201},
		{Name: "server-c", URL: "https://c.io", Method: "GET", Interval: 120, Timeout: 30, ExpectedCode: 200},
	}

	svc := newImportIntegrationService(t)
	file := buildExcel(t, rows)

	result, err := svc.ImportServers(t.Context(), 1, file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 3 {
		t.Errorf("Imported = %d, want 3", result.Imported)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", result.Errors)
	}

	var servers []domain.Server
	if err := testDB.Where("created_by_id = ?", 1).Find(&servers).Error; err != nil {
		t.Fatalf("query servers: %v", err)
	}
	if len(servers) != 3 {
		t.Fatalf("got %d servers in DB, want 3", len(servers))
	}
	names := map[string]bool{}
	for _, s := range servers {
		names[s.Name] = true
	}
	if !names["server-a"] || !names["server-b"] || !names["server-c"] {
		t.Errorf("servers in DB: %v", servers)
	}

	var endpoints []domain.Endpoint
	if err := testDB.Find(&endpoints).Error; err != nil {
		t.Fatalf("query endpoints: %v", err)
	}
	if len(endpoints) != 3 {
		t.Fatalf("got %d endpoints in DB, want 3", len(endpoints))
	}
}

func TestIntegration_ImportServers_SkipEmptyURL(t *testing.T) {
	truncateTables(t)

	rows := []dto.ImportRow{
		{Name: "server-a", URL: "https://a.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		{Name: "server-b", URL: "", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		{Name: "server-c", URL: "https://c.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
	}

	svc := newImportIntegrationService(t)
	file := buildExcel(t, rows)

	result, err := svc.ImportServers(t.Context(), 1, file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 3 {
		t.Errorf("Imported = %d, want 3", result.Imported)
	}

	var servers []domain.Server
	testDB.Find(&servers)
	if len(servers) != 3 {
		t.Fatalf("got %d servers, want 3", len(servers))
	}

	var endpoints []domain.Endpoint
	testDB.Find(&endpoints)
	if len(endpoints) != 2 {
		t.Fatalf("got %d endpoints, want 2", len(endpoints))
	}
}

func TestIntegration_ImportServers_ParseErrors(t *testing.T) {
	truncateTables(t)

	rows := []dto.ImportRow{
		{Name: "", URL: "", Method: "", Interval: 0, Timeout: 0, ExpectedCode: 0},
	}

	svc := newImportIntegrationService(t)
	file := buildExcel(t, rows)

	result, err := svc.ImportServers(t.Context(), 1, file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 0 {
		t.Errorf("Imported = %d, want 0", result.Imported)
	}
	if len(result.Errors) == 0 {
		t.Errorf("expected parse errors, got none")
	}

	var count int64
	testDB.Model(&domain.Server{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 servers in DB, got %d", count)
	}
}

func TestIntegration_ImportServers_PartialErrors(t *testing.T) {
	truncateTables(t)

	rows := []dto.ImportRow{
		{Name: "valid-server", URL: "https://valid.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		{Name: "", URL: "https://bad.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
	}

	svc := newImportIntegrationService(t)
	file := buildExcel(t, rows)

	result, err := svc.ImportServers(t.Context(), 1, file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("Imported = %d, want 1", result.Imported)
	}
	if len(result.Errors) == 0 {
		t.Errorf("expected parse errors, got none")
	}

	var count int64
	testDB.Model(&domain.Server{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 server in DB, got %d", count)
	}
}

func TestIntegration_ImportServers_EmptyFile(t *testing.T) {
	truncateTables(t)

	svc := newImportIntegrationService(t)

	xl := excelize.NewFile()
	buf := new(bytes.Buffer)
	if _, err := xl.WriteTo(buf); err != nil {
		t.Fatalf("write empty excel: %v", err)
	}

	_, err := svc.ImportServers(t.Context(), 1, buf)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestIntegration_ImportServers_DefaultValues(t *testing.T) {
	truncateTables(t)

	svc := newImportIntegrationService(t)

	xl := excelize.NewFile()
	headers := []string{"server_name", "url", "method", "interval_sec", "timeout_sec", "expected_code"}
	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			t.Fatalf("cell name: %v", err)
		}
		if err := xl.SetCellValue("Sheet1", cell, h); err != nil {
			t.Fatalf("set header: %v", err)
		}
	}
	if err := xl.SetCellValue("Sheet1", "A2", "default-server"); err != nil {
		t.Fatalf("set cell: %v", err)
	}
	if err := xl.SetCellValue("Sheet1", "B2", "https://default.com"); err != nil {
		t.Fatalf("set cell: %v", err)
	}
	if err := xl.SetCellValue("Sheet1", "C2", "GET"); err != nil {
		t.Fatalf("set cell: %v", err)
	}

	buf := new(bytes.Buffer)
	if _, err := xl.WriteTo(buf); err != nil {
		t.Fatalf("write excel: %v", err)
	}

	result, err := svc.ImportServers(t.Context(), 1, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("Imported = %d, want 1", result.Imported)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", result.Errors)
	}

	var endpoint domain.Endpoint
	if err := testDB.First(&endpoint).Error; err != nil {
		t.Fatalf("get endpoint: %v", err)
	}
	if endpoint.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", endpoint.Interval)
	}
	if endpoint.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", endpoint.Timeout)
	}
	if endpoint.ExpectedCode != 200 {
		t.Errorf("ExpectedCode = %d, want 200", endpoint.ExpectedCode)
	}
}
