package importer

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/importer/dto"
	monitorrepo "github.com/minhnbnt/uptime-monitor/internal/features/ping/repository"
	"github.com/minhnbnt/uptime-monitor/internal/features/ping/scheduler"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/infrastructure"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

var testDB *gorm.DB
var testRedisAddr string
var testDSN string

func TestMain(m *testing.M) {

	flag.Parse()

	if !testing.Short() {
		ctx := context.Background()

		redisContainer, addr := testcontainers.StartRedisAddr(ctx)
		defer func() { _ = redisContainer.Terminate(ctx) }()
		testRedisAddr = addr
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
			tb.Fatalf("seed test user: %v", err)
		}
	})
}

func newImportIntegrationService(tb testing.TB, redisClient *redis.Client) *ImportService {
	tb.Helper()

	testcontainers.SkipIfShort(tb)

	zsetScheduler := scheduler.NewZSetScheduleRepository(redisClient)
	metaCache := scheduler.NewEndpointMetaCache(redisClient)
	statusStore := monitorrepo.NewRedisServerEventRepository(redisClient)

	return &ImportService{
		serverRepository: serverrepo.NewServerRepository(testDB),
		endpointRepository: serverrepo.NewEndpointRepositoryWithDeps(
			testDB, zsetScheduler, statusStore, metaCache,
		),
		excelExporter: &infrastructure.ExcelExporter{},
		excelParser:   &infrastructure.ExcelParser{},
		logger:        logger.NewMockLogger(),
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
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	redisClient := testcontainers.NewTestRedis(t, testRedisAddr)

	rows := []dto.ImportRow{
		{Name: "server-a", URL: "https://a.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		{Name: "server-b", URL: "https://b.org/ping", Method: "POST", Interval: 60, Timeout: 15, ExpectedCode: 201},
		{Name: "server-c", URL: "https://c.io", Method: "GET", Interval: 120, Timeout: 30, ExpectedCode: 200},
	}

	svc := newImportIntegrationService(t, redisClient)
	file := buildExcel(t, rows)

	result, err := svc.ImportServers(t.Context(), 1, file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Successes) != 3 {
		t.Errorf("len(Successes) = %d, want 3", len(result.Successes))
	}
	if len(result.RowErrors)+len(result.BatchErrors) != 0 {
		t.Errorf("unexpected errors: row=%v batch=%v", result.RowErrors, result.BatchErrors)
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
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	redisClient := testcontainers.NewTestRedis(t, testRedisAddr)

	rows := []dto.ImportRow{
		{Name: "server-a", URL: "https://a.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		{Name: "server-b", URL: "", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		{Name: "server-c", URL: "https://c.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
	}

	svc := newImportIntegrationService(t, redisClient)
	file := buildExcel(t, rows)

	result, err := svc.ImportServers(t.Context(), 1, file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Successes) != 3 {
		t.Errorf("len(Successes) = %d, want 3", len(result.Successes))
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
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	redisClient := testcontainers.NewTestRedis(t, testRedisAddr)

	rows := []dto.ImportRow{
		{Name: "", URL: "", Method: "", Interval: 0, Timeout: 0, ExpectedCode: 0},
	}

	svc := newImportIntegrationService(t, redisClient)
	file := buildExcel(t, rows)

	result, err := svc.ImportServers(t.Context(), 1, file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Successes) != 0 {
		t.Errorf("len(Successes) = %d, want 0", len(result.Successes))
	}
	if len(result.RowErrors) == 0 {
		t.Errorf("expected parse errors, got none")
	}

	var count int64
	testDB.Model(&domain.Server{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 servers in DB, got %d", count)
	}
}

func TestIntegration_ImportServers_PartialErrors(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	redisClient := testcontainers.NewTestRedis(t, testRedisAddr)

	rows := []dto.ImportRow{
		{Name: "valid-server", URL: "https://valid.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		{Name: "", URL: "https://bad.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
	}

	svc := newImportIntegrationService(t, redisClient)
	file := buildExcel(t, rows)

	result, err := svc.ImportServers(t.Context(), 1, file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Successes) != 1 {
		t.Errorf("len(Successes) = %d, want 1", len(result.Successes))
	}
	if len(result.RowErrors) == 0 {
		t.Errorf("expected parse errors, got none")
	}

	var count int64
	testDB.Model(&domain.Server{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 server in DB, got %d", count)
	}
}

func TestIntegration_ImportServers_EmptyFile(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	redisClient := testcontainers.NewTestRedis(t, testRedisAddr)

	svc := newImportIntegrationService(t, redisClient)

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
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	redisClient := testcontainers.NewTestRedis(t, testRedisAddr)

	svc := newImportIntegrationService(t, redisClient)

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
	if len(result.Successes) != 1 {
		t.Errorf("len(Successes) = %d, want 1", len(result.Successes))
	}
	if len(result.RowErrors)+len(result.BatchErrors) != 0 {
		t.Errorf("unexpected errors: row=%v batch=%v", result.RowErrors, result.BatchErrors)
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

func TestIntegration_ImportServers_SchedulerAndCache(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	redisClient := testcontainers.NewTestRedis(t, testRedisAddr)

	rows := []dto.ImportRow{
		{Name: "server-a", URL: "https://a.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		{Name: "server-b", URL: "https://b.org/ping", Method: "POST", Interval: 60, Timeout: 15, ExpectedCode: 201},
		{Name: "server-c", URL: "https://c.io", Method: "GET", Interval: 120, Timeout: 30, ExpectedCode: 200},
	}

	svc := newImportIntegrationService(t, redisClient)
	file := buildExcel(t, rows)

	result, err := svc.ImportServers(t.Context(), 1, file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Successes) != 3 {
		t.Fatalf("len(Successes) = %d, want 3", len(result.Successes))
	}
	if len(result.RowErrors)+len(result.BatchErrors) != 0 {
		t.Fatalf("unexpected errors: row=%v batch=%v", result.RowErrors, result.BatchErrors)
	}

	var endpoints []domain.Endpoint
	if err := testDB.Find(&endpoints).Error; err != nil {
		t.Fatalf("query endpoints: %v", err)
	}
	if len(endpoints) != 3 {
		t.Fatalf("got %d endpoints in DB, want 3", len(endpoints))
	}

	zset := redisClient.ZRangeWithScores(t.Context(), "scheduler:queue", 0, -1)
	zsetResult, err := zset.Result()
	if err != nil {
		t.Fatalf("ZRANGE scheduler:queue: %v", err)
	}
	if len(zsetResult) != 3 {
		t.Fatalf("got %d entries in scheduler:queue, want 3", len(zsetResult))
	}

	epIDs := make(map[uint]bool)
	for _, ep := range endpoints {
		epIDs[ep.ID] = true
	}

	for _, z := range zsetResult {
		member, ok := z.Member.(string)
		if !ok {
			t.Errorf("expected string member, got %T", z.Member)
			continue
		}
		var id uint
		if _, err := fmt.Sscanf(member, "%d", &id); err != nil {
			t.Errorf("parse member %q: %v", member, err)
			continue
		}
		if !epIDs[id] {
			t.Errorf("unexpected endpoint %d in scheduler:queue", id)
		}
	}

	for _, ep := range endpoints {
		metaKey := fmt.Sprintf("scheduler:meta:%d", ep.ID)
		data, err := redisClient.HGetAll(t.Context(), metaKey).Result()
		if err != nil {
			t.Fatalf("HGetAll %s: %v", metaKey, err)
		}
		if len(data) == 0 {
			t.Errorf("meta cache key %s is empty", metaKey)
			continue
		}
		if data["url"] != ep.URL {
			t.Errorf("meta cache url = %q, want %q", data["url"], ep.URL)
		}
		if data["method"] != ep.Method {
			t.Errorf("meta cache method = %q, want %q", data["method"], ep.Method)
		}
		if data["expected_code"] != fmt.Sprint(ep.ExpectedCode) {
			t.Errorf("meta cache expected_code = %q, want %d", data["expected_code"], ep.ExpectedCode)
		}
		if data["interval_ns"] != fmt.Sprint(ep.Interval.Nanoseconds()) {
			t.Errorf("meta cache interval_ns = %q, want %d", data["interval_ns"], ep.Interval.Nanoseconds())
		}
	}
}

func TestIntegration_ImportServers_EmptyURL_NoScheduler(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	redisClient := testcontainers.NewTestRedis(t, testRedisAddr)

	rows := []dto.ImportRow{
		{Name: "server-a", URL: "https://a.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		{Name: "server-b", URL: "", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		{Name: "server-c", URL: "https://c.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
	}

	svc := newImportIntegrationService(t, redisClient)
	file := buildExcel(t, rows)

	result, err := svc.ImportServers(t.Context(), 1, file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Successes) != 3 {
		t.Errorf("len(Successes) = %d, want 3", len(result.Successes))
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

	zset := redisClient.ZCard(t.Context(), "scheduler:queue")
	if zset.Val() != 2 {
		t.Errorf("got %d entries in scheduler:queue, want 2", zset.Val())
	}

	var metaCount int
	for _, ep := range endpoints {
		metaKey := fmt.Sprintf("scheduler:meta:%d", ep.ID)
		exists, err := redisClient.Exists(t.Context(), metaKey).Result()
		if err != nil {
			t.Fatalf("Exists %s: %v", metaKey, err)
		}
		if exists == 1 {
			metaCount++
		}
	}
	if metaCount != 2 {
		t.Errorf("got %d meta cache entries, want 2", metaCount)
	}
}

func TestIntegration_ImportServers_MetaCacheLookup(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	redisClient := testcontainers.NewTestRedis(t, testRedisAddr)

	rows := []dto.ImportRow{
		{Name: "server-a", URL: "https://a.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
	}

	svc := newImportIntegrationService(t, redisClient)
	file := buildExcel(t, rows)

	result, err := svc.ImportServers(t.Context(), 1, file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Successes) != 1 {
		t.Fatalf("len(Successes) = %d, want 1", len(result.Successes))
	}

	var endpoint domain.Endpoint
	if err := testDB.First(&endpoint).Error; err != nil {
		t.Fatalf("get endpoint: %v", err)
	}

	metaCache := scheduler.NewEndpointMetaCache(redisClient)

	cached, err := metaCache.Get(t.Context(), endpoint.ID)
	if err != nil {
		t.Fatalf("metaCache.Get: %v", err)
	}
	if cached.URL != endpoint.URL {
		t.Errorf("cached URL = %q, want %q", cached.URL, endpoint.URL)
	}
	if cached.Method != endpoint.Method {
		t.Errorf("cached Method = %q, want %q", cached.Method, endpoint.Method)
	}
	if cached.ExpectedCode != endpoint.ExpectedCode {
		t.Errorf("cached ExpectedCode = %d, want %d", cached.ExpectedCode, endpoint.ExpectedCode)
	}
	if cached.Interval != endpoint.Interval {
		t.Errorf("cached Interval = %v, want %v", cached.Interval, endpoint.Interval)
	}

	lookupEndpoint, err := metaCache.Get(t.Context(), endpoint.ID)
	if err != nil {
		t.Fatalf("second metaCache.Get: %v", err)
	}
	if lookupEndpoint == nil {
		t.Fatal("lookup endpoint is nil")
	}
}
