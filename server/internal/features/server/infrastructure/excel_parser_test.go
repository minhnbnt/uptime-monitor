package infrastructure

import (
	"bytes"
	"io"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

func newTestFile(t *testing.T, rows [][]string) io.Reader {
	t.Helper()
	xl := excelize.NewFile()
	for i, row := range rows {
		for j, cell := range row {
			cellName, err := excelize.CoordinatesToCellName(j+1, i+1)
			if err != nil {
				t.Fatalf("coordinates: %v", err)
			}
			if err := xl.SetCellValue("Sheet1", cellName, cell); err != nil {
				t.Fatalf("set cell: %v", err)
			}
		}
	}
	buf := &bytes.Buffer{}
	if err := xl.Write(buf); err != nil {
		t.Fatalf("write xlsx: %v", err)
	}
	return buf
}

func TestGetCellByHeader(t *testing.T) {
	row := []string{"a", "b", "c"}
	colMap := map[string]int{"name": 0, "url": 2}
	if got := getCellByHeader(row, colMap, "name"); got != "a" {
		t.Errorf("getCellByHeader(row, colMap, \"name\") = %q, want %q", got, "a")
	}
	if got := getCellByHeader(row, colMap, "url"); got != "c" {
		t.Errorf("getCellByHeader(row, colMap, \"url\") = %q, want %q", got, "c")
	}
	if got := getCellByHeader(row, colMap, "missing"); got != "" {
		t.Errorf("getCellByHeader(row, colMap, \"missing\") = %q, want empty", got)
	}
}

func TestParseServerName(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		v, err := parseServerName(" My Server ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "My Server" {
			t.Errorf("got %q, want %q", v, "My Server")
		}
	})
	t.Run("empty", func(t *testing.T) {
		_, err := parseServerName("")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestParseURL(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		v, err := parseURL(" https://example.com ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "https://example.com" {
			t.Errorf("got %q, want %q", v, "https://example.com")
		}
	})
	t.Run("empty", func(t *testing.T) {
		v, err := parseURL("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "" {
			t.Errorf("got %q, want empty", v)
		}
	})
}

func TestParseMethod(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		v, err := parseMethod("GET")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "GET" {
			t.Errorf("got %q, want %q", v, "GET")
		}
	})
	t.Run("default", func(t *testing.T) {
		v, err := parseMethod("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "GET" {
			t.Errorf("got %q, want %q", v, "GET")
		}
	})
}

func TestParseInterval(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		v, err := parseInterval("60")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 60 {
			t.Errorf("got %d, want %d", v, 60)
		}
	})
	t.Run("default", func(t *testing.T) {
		v, err := parseInterval("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 30 {
			t.Errorf("got %d, want %d", v, 30)
		}
	})
	t.Run("out of range", func(t *testing.T) {
		_, err := parseInterval("-1")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestParseTimeout(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		v, err := parseTimeout("15")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 15 {
			t.Errorf("got %d, want %d", v, 15)
		}
	})
	t.Run("default", func(t *testing.T) {
		v, err := parseTimeout("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 10 {
			t.Errorf("got %d, want %d", v, 10)
		}
	})
	t.Run("out of range", func(t *testing.T) {
		_, err := parseTimeout("0")
		if err != utils.ErrTimeoutInvalid {
			t.Errorf("got %v, want %v", err, utils.ErrTimeoutInvalid)
		}
	})
}

func TestParseExpectedCode(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		v, err := parseExpectedCode("201")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 201 {
			t.Errorf("got %d, want %d", v, 201)
		}
	})
	t.Run("default", func(t *testing.T) {
		v, err := parseExpectedCode("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 200 {
			t.Errorf("got %d, want %d", v, 200)
		}
	})
	t.Run("out of range", func(t *testing.T) {
		_, err := parseExpectedCode("600")
		if err != utils.ErrCodeOutOfRange {
			t.Errorf("got %v, want %v", err, utils.ErrCodeOutOfRange)
		}
	})
}

func TestParseRow(t *testing.T) {
	colMap := map[string]int{
		"server_name":   0,
		"url":           1,
		"method":        2,
		"interval_sec":  3,
		"timeout_sec":   4,
		"expected_code": 5,
	}

	t.Run("valid row", func(t *testing.T) {
		row := []string{"My Server", "https://example.com", "GET", "30", "10", "200"}
		parsed, errs := parseRow(2, row, colMap)
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		if parsed.Name != "My Server" || parsed.URL != "https://example.com" || parsed.Method != "GET" {
			t.Errorf("parsed = %+v", parsed)
		}
	})
	t.Run("row with empty optional fields", func(t *testing.T) {
		row := []string{"My Server", "", "", "", "", ""}
		parsed, errs := parseRow(3, row, colMap)
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		if parsed.Interval != 30 || parsed.Timeout != 10 || parsed.ExpectedCode != 200 || parsed.Method != "GET" {
			t.Errorf("defaults not applied: %+v", parsed)
		}
	})
	t.Run("row with errors", func(t *testing.T) {
		row := []string{"", "invalid", "INVALID", "-1", "0", "999"}
		_, errs := parseRow(4, row, colMap)
		if len(errs) == 0 {
			t.Fatal("expected errors, got none")
		}
	})
	t.Run("short row", func(t *testing.T) {
		row := []string{"My Server"}
		parsed, errs := parseRow(5, row, colMap)
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		if parsed.Name != "My Server" {
			t.Errorf("Name = %q", parsed.Name)
		}
	})
}

func TestParseImportFile(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		f := newTestFile(t, [][]string{
			{"server_name", "url", "method", "interval_sec", "timeout_sec", "expected_code"},
			{"Server A", "https://a.com", "GET", "30", "10", "200"},
			{"Server B", "https://b.com", "POST", "60", "15", "201"},
		})
		rows, errs, err := (&ExcelParser{}).ParseImportFile(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(errs) != 0 {
			t.Fatalf("unexpected row errors: %v", errs)
		}
		if len(rows) != 2 {
			t.Fatalf("got %d rows, want 2", len(rows))
		}
	})
	t.Run("file with parse errors", func(t *testing.T) {
		f := newTestFile(t, [][]string{
			{"server_name", "url", "method", "interval_sec", "timeout_sec", "expected_code"},
			{"", "invalid://url", "INVALID", "abc", "def", "xyz"},
		})
		rows, errs, err := (&ExcelParser{}).ParseImportFile(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rows) != 0 {
			t.Errorf("expect 0 valid rows, got %d", len(rows))
		}
		if len(errs) == 0 {
			t.Fatal("expected row errors, got none")
		}
	})
	t.Run("no data rows", func(t *testing.T) {
		f := newTestFile(t, [][]string{
			{"server_name", "url"},
		})
		_, _, err := (&ExcelParser{}).ParseImportFile(f)
		if err == nil {
			t.Fatal("expected error for no data rows")
		}
	})
	t.Run("invalid file format", func(t *testing.T) {
		_, _, err := (&ExcelParser{}).ParseImportFile(bytes.NewReader([]byte("not an xlsx file")))
		if err == nil {
			t.Fatal("expected error for invalid file")
		}
	})
	t.Run("missing headers", func(t *testing.T) {
		f := newTestFile(t, [][]string{
			{"server_name", "url"},
			{"Server A", "https://a.com"},
		})
		_, _, err := (&ExcelParser{}).ParseImportFile(f)
		if err == nil {
			t.Fatal("expected error for missing headers")
		}
	})
}
