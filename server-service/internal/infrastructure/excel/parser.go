package excel

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/samber/lo/it"
	"github.com/xuri/excelize/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/utils"
)

type ExcelParser struct{}

func RegisterExcelParser(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ExcelParser, error) {
		return &ExcelParser{}, nil
	})
}

func (p *ExcelParser) ParseImportFile(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error) {

	xl, err := excelize.OpenReader(file)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid excel file: %w", err)
	}

	defer func() { _ = xl.Close() }()

	rows, err := xl.GetRows(xl.GetSheetName(0))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read sheet: %w", err)
	}
	if len(rows) <= 1 {
		return nil, nil, fmt.Errorf("excel file has no data rows")
	}

	colMap := buildColumnMap(rows[0])
	if err := validateHeaders(colMap); err != nil {
		return nil, nil, err
	}

	valid, rowErrors := []dto.ImportRow{}, []dto.ImportRowError{}

	for i, row := range rows[1:] {

		rowNum := i + 2
		parsed, errs := parseRow(rowNum, row, colMap)
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

var expectedHeaders = []string{
	"server_name",
	"url", "method",
	"interval_sec",
	"timeout_sec",
	"expected_code",
}

func validateHeaders(colMap map[string]int) error {

	missing := lo.Filter(expectedHeaders, func(h string, _ int) bool {
		_, ok := colMap[h]
		return !ok
	})

	if len(missing) > 0 {
		return fmt.Errorf("missing required column(s): %s", strings.Join(missing, ", "))
	}

	return nil
}

func buildColumnMap(headers []string) map[string]int {

	m := make(map[string]int, len(headers))
	for i, h := range headers {
		m[strings.TrimSpace(h)] = i
	}

	return m
}

func getCellByHeader(row []string, colMap map[string]int, header string) string {

	idx, ok := colMap[header]
	if !ok {
		return ""
	}

	if idx >= len(row) {
		return ""
	}

	return row[idx]
}

func parseRow(rowNum int, row []string, colMap map[string]int) (dto.ImportRow, []error) {

	errs, err := []error{}, error(nil)
	r := dto.ImportRow{}

	r.Row = rowNum

	r.Name, err = parseServerName(getCellByHeader(row, colMap, "server_name"))
	if err != nil {
		errs = append(errs, err)
	}

	r.URL, err = parseURL(getCellByHeader(row, colMap, "url"))
	if err != nil {
		errs = append(errs, err)
	}

	r.Method, err = parseMethod(getCellByHeader(row, colMap, "method"))
	if err != nil {
		errs = append(errs, err)
	}

	r.Interval, err = parseInterval(getCellByHeader(row, colMap, "interval_sec"))
	if err != nil {
		errs = append(errs, err)
	}

	r.Timeout, err = parseTimeout(getCellByHeader(row, colMap, "timeout_sec"))
	if err != nil {
		errs = append(errs, err)
	}

	r.ExpectedCode, err = parseExpectedCode(getCellByHeader(row, colMap, "expected_code"))
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
