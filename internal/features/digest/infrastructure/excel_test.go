package infrastructure

import (
	"io"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

func defaultSummary() ServerSummary {
	return ServerSummary{Total: 10, Online: 7, Offline: 3}
}

func TestGenerateStatusReport_SheetNameConstant(t *testing.T) {
	if ReportSheetName != "Server Uptime" {
		t.Errorf("ReportSheetName = %q, want Server Uptime", ReportSheetName)
	}
}

func readSheetRows(t *testing.T, r io.Reader, sheet string) [][]string {
	t.Helper()
	xl, err := excelize.OpenReader(r)
	if err != nil {
		t.Fatalf("not a valid xlsx: %v", err)
	}
	defer xl.Close()
	rows, err := xl.GetRows(sheet)
	if err != nil {
		t.Fatalf("get rows from %s: %v", sheet, err)
	}
	return rows
}

func TestGenerateStatusReport_Empty(t *testing.T) {
	r, err := GenerateStatusReport(nil, nil, defaultSummary())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows := readSheetRows(t, r, ReportSheetName)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (header), got %d", len(rows))
	}
	if rows[0][0] != "Server Name" {
		t.Errorf("header[0] = %q, want 'Server Name'", rows[0][0])
	}
}

func TestGenerateStatusReport_WithDates(t *testing.T) {
	d1 := utils.TruncateDay(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	d2 := utils.TruncateDay(time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC))
	r, err := GenerateStatusReport(nil, []time.Time{d1, d2}, defaultSummary())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows := readSheetRows(t, r, ReportSheetName)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if len(rows[0]) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(rows[0]))
	}
	if rows[0][1] != "2025-06-01" || rows[0][2] != "2025-06-02" {
		t.Errorf("dates: %v", rows[0][1:])
	}
}

func TestGenerateStatusReport_SingleRow(t *testing.T) {
	d1 := utils.TruncateDay(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	d2 := utils.TruncateDay(time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC))
	rows := []ServerRow{
		{ServerID: 1, ServerName: "Alpha", Stats: map[time.Time]float64{d1: 99.5, d2: 87.3}},
	}
	r, err := GenerateStatusReport(rows, []time.Time{d1, d2}, defaultSummary())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readSheetRows(t, r, ReportSheetName)
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
	if got[1][0] != "Alpha" {
		t.Errorf("name = %q, want Alpha", got[1][0])
	}
	if got[1][1] != "99.50%" {
		t.Errorf("d1 = %q, want 99.50%%", got[1][1])
	}
	if got[1][2] != "87.30%" {
		t.Errorf("d2 = %q, want 87.30%%", got[1][2])
	}
}

func TestGenerateStatusReport_MissingStats(t *testing.T) {
	d1 := utils.TruncateDay(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	d2 := utils.TruncateDay(time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC))
	rows := []ServerRow{
		{ServerID: 1, ServerName: "Beta", Stats: map[time.Time]float64{d1: 95.0}},
	}
	r, err := GenerateStatusReport(rows, []time.Time{d1, d2}, defaultSummary())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readSheetRows(t, r, ReportSheetName)
	if got[1][1] != "95.00%" {
		t.Errorf("d1 = %q, want 95.00%%", got[1][1])
	}
	if got[1][2] != "-" {
		t.Errorf("d2 = %q, want '-'", got[1][2])
	}
}

func TestGenerateStatusReport_MultipleServers(t *testing.T) {
	d := utils.TruncateDay(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	rows := []ServerRow{
		{ServerID: 1, ServerName: "Alpha", Stats: map[time.Time]float64{d: 100}},
		{ServerID: 2, ServerName: "Beta", Stats: map[time.Time]float64{d: 50.5}},
	}
	r, err := GenerateStatusReport(rows, []time.Time{d}, defaultSummary())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readSheetRows(t, r, ReportSheetName)
	if len(got) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(got))
	}
	if got[1][0] != "Alpha" || got[2][0] != "Beta" {
		t.Errorf("names: %q, %q", got[1][0], got[2][0])
	}
}

func TestGenerateStatusReport_SummarySheet(t *testing.T) {
	summary := ServerSummary{Total: 15, Online: 10, Offline: 5}
	r, err := GenerateStatusReport(nil, nil, summary)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows := readSheetRows(t, r, SummarySheetName)
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows (header + 3 data), got %d", len(rows))
	}
	if rows[0][0] != "Metric" || rows[0][1] != "Count" {
		t.Errorf("header = %v, want [Metric Count]", rows[0])
	}
	if rows[1][0] != "Total Servers" || rows[1][1] != "15" {
		t.Errorf("row 1 = %v, want [Total Servers 15]", rows[1])
	}
	if rows[2][0] != "Online" || rows[2][1] != "10" {
		t.Errorf("row 2 = %v, want [Online 10]", rows[2])
	}
	if rows[3][0] != "Offline" || rows[3][1] != "5" {
		t.Errorf("row 3 = %v, want [Offline 5]", rows[3])
	}
}
