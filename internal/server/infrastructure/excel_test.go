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

func TestGetCell(t *testing.T) {
	row := []string{"a", "b", "c"}
	if got := getCell(row, 0); got != "a" {
		t.Errorf("getCell(row, 0) = %q, want %q", got, "a")
	}
	if got := getCell(row, 2); got != "c" {
		t.Errorf("getCell(row, 2) = %q, want %q", got, "c")
	}
	if got := getCell(row, 3); got != "" {
		t.Errorf("getCell(row, 3) = %q, want empty", got)
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
		if err != utils.ErrNameRequired {
			t.Errorf("got %v, want %v", err, utils.ErrNameRequired)
		}
	})
	t.Run("too long", func(t *testing.T) {
		long := make([]byte, 256)
		for i := range long {
			long[i] = 'a'
		}
		_, err := parseServerName(string(long))
		if err != utils.ErrNameTooLong {
			t.Errorf("got %v, want %v", err, utils.ErrNameTooLong)
		}
	})
}

func TestParseURL(t *testing.T) {
	t.Run("valid http", func(t *testing.T) {
		v, err := parseURL("http://example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "http://example.com" {
			t.Errorf("got %q", v)
		}
	})
	t.Run("valid https", func(t *testing.T) {
		v, err := parseURL(" https://example.com/path ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "https://example.com/path" {
			t.Errorf("got %q", v)
		}
	})
	t.Run("empty is valid", func(t *testing.T) {
		v, err := parseURL("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "" {
			t.Errorf("got %q, want empty", v)
		}
	})
	t.Run("invalid scheme", func(t *testing.T) {
		_, err := parseURL("ftp://example.com")
		if err != utils.ErrURLInvalid {
			t.Errorf("got %v, want %v", err, utils.ErrURLInvalid)
		}
	})
}

func TestParseMethod(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		v, err := parseMethod("POST")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "POST" {
			t.Errorf("got %q, want %q", v, "POST")
		}
	})
	t.Run("case insensitive", func(t *testing.T) {
		v, err := parseMethod("get")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "GET" {
			t.Errorf("got %q, want %q", v, "GET")
		}
	})
	t.Run("empty defaults to GET", func(t *testing.T) {
		v, err := parseMethod("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "GET" {
			t.Errorf("got %q, want %q", v, "GET")
		}
	})
	t.Run("unsupported", func(t *testing.T) {
		_, err := parseMethod("INVALID")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestParseInterval(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		v, err := parseInterval(" 60 ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 60 {
			t.Errorf("got %d, want %d", v, 60)
		}
	})
	t.Run("empty defaults to 30", func(t *testing.T) {
		v, err := parseInterval("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 30 {
			t.Errorf("got %d, want %d", v, 30)
		}
	})
	t.Run("invalid int", func(t *testing.T) {
		_, err := parseInterval("abc")
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("negative", func(t *testing.T) {
		_, err := parseInterval("-5")
		if err != utils.ErrIntervalInvalid {
			t.Errorf("got %v, want %v", err, utils.ErrIntervalInvalid)
		}
	})
}

func TestParseTimeout(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		v, err := parseTimeout(" 15 ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 15 {
			t.Errorf("got %d, want %d", v, 15)
		}
	})
	t.Run("empty defaults to 10", func(t *testing.T) {
		v, err := parseTimeout("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 10 {
			t.Errorf("got %d, want %d", v, 10)
		}
	})
	t.Run("invalid int", func(t *testing.T) {
		_, err := parseTimeout("abc")
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("negative", func(t *testing.T) {
		_, err := parseTimeout("-1")
		if err != utils.ErrTimeoutInvalid {
			t.Errorf("got %v, want %v", err, utils.ErrTimeoutInvalid)
		}
	})
}

func TestParseExpectedCode(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		v, err := parseExpectedCode(" 301 ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 301 {
			t.Errorf("got %d, want %d", v, 301)
		}
	})
	t.Run("empty defaults to 200", func(t *testing.T) {
		v, err := parseExpectedCode("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 200 {
			t.Errorf("got %d, want %d", v, 200)
		}
	})
	t.Run("invalid int", func(t *testing.T) {
		_, err := parseExpectedCode("abc")
		if err == nil {
			t.Fatal("expected error")
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
	t.Run("valid row", func(t *testing.T) {
		row := []string{"My Server", "https://example.com", "GET", "30", "10", "200"}
		parsed, errs := parseRow(2, row)
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		if parsed.Name != "My Server" || parsed.URL != "https://example.com" || parsed.Method != "GET" {
			t.Errorf("parsed = %+v", parsed)
		}
	})
	t.Run("row with empty optional fields", func(t *testing.T) {
		row := []string{"My Server", "", "", "", "", ""}
		parsed, errs := parseRow(3, row)
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		if parsed.Interval != 30 || parsed.Timeout != 10 || parsed.ExpectedCode != 200 || parsed.Method != "GET" {
			t.Errorf("defaults not applied: %+v", parsed)
		}
	})
	t.Run("row with errors", func(t *testing.T) {
		row := []string{"", "invalid", "INVALID", "-1", "0", "999"}
		_, errs := parseRow(4, row)
		if len(errs) == 0 {
			t.Fatal("expected errors, got none")
		}
	})
	t.Run("short row", func(t *testing.T) {
		row := []string{"My Server"}
		parsed, errs := parseRow(5, row)
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
		rows, errs, err := (&ExcelGenerator{}).ParseImportFile(f)
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
		rows, errs, err := (&ExcelGenerator{}).ParseImportFile(f)
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
		_, _, err := (&ExcelGenerator{}).ParseImportFile(f)
		if err == nil {
			t.Fatal("expected error for no data rows")
		}
	})
	t.Run("invalid file format", func(t *testing.T) {
		_, _, err := (&ExcelGenerator{}).ParseImportFile(bytes.NewReader([]byte("not an xlsx file")))
		if err == nil {
			t.Fatal("expected error for invalid file")
		}
	})
}

func TestGenerateTemplate(t *testing.T) {
	t.Run("writes valid xlsx", func(t *testing.T) {
		var buf bytes.Buffer
		err := (&ExcelGenerator{}).GenerateTemplate(&buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.Len() == 0 {
			t.Fatal("empty output")
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
		if len(rows) < 2 {
			t.Fatalf("expected at least 2 rows, got %d", len(rows))
		}
		headers := []string{"server_name", "url", "method", "interval_sec", "timeout_sec", "expected_code"}
		for i, h := range headers {
			if rows[0][i] != h {
				t.Errorf("header[%d] = %q, want %q", i, rows[0][i], h)
			}
		}
		if rows[1][0] != "My Server" {
			t.Errorf("example row server_name = %q", rows[1][0])
		}
		if rows[1][1] != "https://example.com/health" {
			t.Errorf("example row url = %q", rows[1][1])
		}
	})
}
