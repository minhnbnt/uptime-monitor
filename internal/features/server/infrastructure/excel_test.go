package infrastructure

import (
	"bytes"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	serverdto "github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

func TestGenerateExportFile_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := (&ExcelExporter{}).GenerateExportFile(&buf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xl, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatalf("not a valid xlsx: %v", err)
	}
	defer xl.Close()
	rows, err := xl.GetRows("Sheet1")
	if err != nil {
		t.Fatalf("get rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (header only), got %d", len(rows))
	}
	expected := []string{"server_name", "url", "method", "interval_sec", "timeout_sec", "expected_code", "status"}
	for i, h := range expected {
		if rows[0][i] != h {
			t.Errorf("header[%d] = %q, want %q", i, rows[0][i], h)
		}
	}
}

func TestGenerateExportFile_SingleServerOnline(t *testing.T) {
	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	servers := []serverdto.Server{
		{
			Name:          "Test Server",
			MonitorStatus: domain.StatusOn,
			Endpoint:      &serverdto.Endpoint{URL: "https://example.com/health", Method: "GET", Interval: 30 * time.Second, Timeout: 10 * time.Second, ExpectedCode: 200},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
	err := (&ExcelExporter{}).GenerateExportFile(&buf, servers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xl, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatalf("not a valid xlsx: %v", err)
	}
	defer xl.Close()
	rows, err := xl.GetRows("Sheet1")
	if err != nil {
		t.Fatalf("get rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[1][1] != "https://example.com/health" {
		t.Errorf("url = %q", rows[1][1])
	}
	if rows[1][2] != "GET" {
		t.Errorf("method = %q", rows[1][2])
	}
	if rows[1][3] != "30" {
		t.Errorf("interval_sec = %q", rows[1][3])
	}
	if rows[1][4] != "10" {
		t.Errorf("timeout_sec = %q", rows[1][4])
	}
	if rows[1][5] != "200" {
		t.Errorf("expected_code = %q", rows[1][5])
	}
	if rows[1][6] != "online" {
		t.Errorf("status = %q, want online", rows[1][6])
	}
}

func TestGenerateExportFile_SingleServerOfflineNoEndpoint(t *testing.T) {
	var buf bytes.Buffer
	servers := []serverdto.Server{
		{
			Name:          "Offline Server",
			MonitorStatus: domain.StatusOff,
			Endpoint:      nil,
		},
	}
	err := (&ExcelExporter{}).GenerateExportFile(&buf, servers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xl, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatalf("not a valid xlsx: %v", err)
	}
	defer xl.Close()
	rows, err := xl.GetRows("Sheet1")
	if err != nil {
		t.Fatalf("get rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[1][1] != "" {
		t.Errorf("url = %q, want empty", rows[1][1])
	}
	if rows[1][2] != "GET" {
		t.Errorf("method = %q, want GET", rows[1][2])
	}
	if rows[1][3] != "30" {
		t.Errorf("interval_sec = %q, want 30", rows[1][3])
	}
	if rows[1][4] != "10" {
		t.Errorf("timeout_sec = %q, want 10", rows[1][4])
	}
	if rows[1][5] != "200" {
		t.Errorf("expected_code = %q, want 200", rows[1][5])
	}
	if rows[1][6] != "offline" {
		t.Errorf("status = %q, want offline", rows[1][6])
	}
}

func TestGenerateExportFile_MultipleServers(t *testing.T) {
	var buf bytes.Buffer
	servers := []serverdto.Server{
		{Name: "Alpha", MonitorStatus: domain.StatusOn, Endpoint: &serverdto.Endpoint{URL: "https://a.com", Method: "GET", Interval: 30 * time.Second, Timeout: 10 * time.Second, ExpectedCode: 200}},
		{Name: "Beta", MonitorStatus: domain.StatusOff, Endpoint: nil},
	}
	err := (&ExcelExporter{}).GenerateExportFile(&buf, servers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xl, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatalf("not a valid xlsx: %v", err)
	}
	defer xl.Close()
	rows, err := xl.GetRows("Sheet1")
	if err != nil {
		t.Fatalf("get rows: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[1][0] != "Alpha" || rows[2][0] != "Beta" {
		t.Errorf("names: %q, %q", rows[1][0], rows[2][0])
	}
	expected := []string{"server_name", "url", "method", "interval_sec", "timeout_sec", "expected_code", "status"}
	for i, h := range expected {
		if rows[0][i] != h {
			t.Errorf("header[%d] = %q, want %q", i, rows[0][i], h)
		}
	}
}

func TestGenerateTemplate(t *testing.T) {
	t.Run("writes valid xlsx", func(t *testing.T) {
		var buf bytes.Buffer
		err := (&ExcelExporter{}).GenerateTemplate(&buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		xl, err := excelize.OpenReader(&buf)
		if err != nil {
			t.Fatalf("not a valid xlsx: %v", err)
		}
		xl.Close()
	})
}
