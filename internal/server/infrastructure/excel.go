package infrastructure

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo/it"
	"github.com/xuri/excelize/v2"

	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

type ExcelGenerator struct{}

func RegisterExcelGenerator(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ExcelGenerator, error) {
		return &ExcelGenerator{}, nil
	})
}

func (g *ExcelGenerator) GenerateTemplate(w io.Writer) error {

	xl := excelize.NewFile()
	defer xl.Close()

	headers := []string{
		"server_name",
		"url", "method",
		"interval_sec",
		"timeout_sec",
		"expected_code",
	}

	if err := setHeader(xl, "Sheet1", headers); err != nil {
		return fmt.Errorf("failed to set header: %w", err)
	}

	if err := xl.SetCellValue("Sheet1", "A2", "My Server"); err != nil {
		return fmt.Errorf("failed to set cell value: %w", err)
	}
	if err := xl.SetCellValue("Sheet1", "B2", "https://example.com/health"); err != nil {
		return fmt.Errorf("failed to set cell value: %w", err)
	}

	if err := xl.Write(w); err != nil {
		return fmt.Errorf("failed to write Excel file: %w", err)
	}

	return nil
}

func setHeader(f *excelize.File, sheet string, headers []string) error {

	for i, h := range headers {

		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return fmt.Errorf("failed to create cell name: %w", err)
		}

		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
	}

	return nil
}

func (g *ExcelGenerator) GenerateExportFile(w io.Writer, servers []dto.Server) error {

	xl := excelize.NewFile()
	defer xl.Close()

	headers := []string{
		"server_name",
		"status",
		"endpoint_url",
		"created_at",
		"updated_at",
	}
	if err := setHeader(xl, "Sheet1", headers); err != nil {
		return fmt.Errorf("failed to set header: %w", err)
	}

	for i, sv := range servers {

		url := ""

		if sv.Endpoint != nil {
			url = sv.Endpoint.URL
		}

		row := i + 2
		values := map[string]string{
			fmt.Sprintf("A%d", row): sv.Name,
			fmt.Sprintf("B%d", row): string(sv.Status),
			fmt.Sprintf("C%d", row): url,
			fmt.Sprintf("D%d", row): sv.CreatedAt.Format(time.RFC3339),
			fmt.Sprintf("E%d", row): sv.UpdatedAt.Format(time.RFC3339),
		}

		for cell, value := range values {
			if err := xl.SetCellValue("Sheet1", cell, value); err != nil {
				return fmt.Errorf("failed to set cell value: %w", err)
			}
		}
	}

	if err := xl.Write(w); err != nil {
		return fmt.Errorf("failed to write Excel file: %w", err)
	}

	return nil
}

func (g *ExcelGenerator) ParseImportFile(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error) {

	xl, err := excelize.OpenReader(file)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid excel file: %w", err)
	}

	defer xl.Close()

	rows, err := xl.GetRows(xl.GetSheetName(0))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read sheet: %w", err)
	}
	if len(rows) <= 1 {
		return nil, nil, fmt.Errorf("excel file has no data rows")
	}

	valid, rowErrors := []dto.ImportRow{}, []dto.ImportRowError{}

	for i, row := range rows[1:] {

		rowNum := i + 2
		parsed, errs := parseRow(rowNum, row)
		if len(errs) == 0 {
			valid = append(valid, parsed)
			continue
		}

		errsIter := slices.Values(errs)
		newErrors := it.Map(errsIter, func(err error) dto.ImportRowError {
			return dto.ImportRowError{
				Row:     rowNum,
				Message: err.Error(),
			}
		})

		rowErrors = slices.AppendSeq(rowErrors, newErrors)
	}

	return valid, rowErrors, nil
}

func parseRow(rowNum int, row []string) (dto.ImportRow, []error) {

	errs, err := []error{}, error(nil)
	r := dto.ImportRow{}

	r.Row = rowNum

	r.Name, err = parseServerName(getCell(row, 0))
	if err != nil {
		errs = append(errs, err)
	}

	r.URL, err = parseURL(getCell(row, 1))
	if err != nil {
		errs = append(errs, err)
	}

	r.Method, err = parseMethod(getCell(row, 2))
	if err != nil {
		errs = append(errs, err)
	}

	r.Interval, err = parseInterval(getCell(row, 3))
	if err != nil {
		errs = append(errs, err)
	}

	r.Timeout, err = parseTimeout(getCell(row, 4))
	if err != nil {
		errs = append(errs, err)
	}

	r.ExpectedCode, err = parseExpectedCode(getCell(row, 5))
	if err != nil {
		errs = append(errs, err)
	}

	return r, errs
}

func parseServerName(v string) (string, error) {

	v = strings.TrimSpace(v)
	if err := utils.ValidateServerName(v); err != nil {
		return v, err
	}

	return v, nil
}

func parseURL(v string) (string, error) {

	if err := utils.ValidateURL(v); err != nil {
		return "", err
	}

	return strings.TrimSpace(v), nil
}

func parseMethod(v string) (string, error) {
	return utils.ValidateMethod(v)
}

func parseIntCell(v, field string, defaultValue int, validateFn func(int) error) (int, error) {

	v = strings.TrimSpace(v)
	if v == "" {
		return defaultValue, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer", field)
	}

	if err := validateFn(n); err != nil {
		return 0, err
	}

	return n, nil
}

func parseInterval(v string) (int, error) {
	return parseIntCell(v, "interval_sec", 30, utils.ValidateInterval)
}

func parseTimeout(v string) (int, error) {
	return parseIntCell(v, "timeout_sec", 10, utils.ValidateTimeout)
}

func parseExpectedCode(v string) (int, error) {
	return parseIntCell(v, "expected_code", 200, utils.ValidateExpectedCode)
}

func getCell(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return row[idx]
}
