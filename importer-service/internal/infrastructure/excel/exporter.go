package excel

import (
	"fmt"
	"io"

	"github.com/samber/do/v2"
	"github.com/xuri/excelize/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/domain"
	serverdto "github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/utils"
)

type ExcelExporter struct{}

func RegisterExcelExporter(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ExcelExporter, error) {
		return &ExcelExporter{}, nil
	})
}

func fillTemplate(xl *excelize.File) error {

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

	return nil
}

func (g *ExcelExporter) GenerateTemplate() (io.ReadCloser, error) {

	xl := excelize.NewFile()

	if err := fillTemplate(xl); err != nil {
		_ = xl.Close()
		return nil, fmt.Errorf("failed to fill template: %w", err)
	}

	pr, pw := io.Pipe()
	go func() {

		defer xl.Close()

		if err := xl.Write(pw); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to write Excel file: %w", err))
		} else {
			pw.Close()
		}

	}()

	return pr, nil
}

func fillExportFile(xl *excelize.File, servers []serverdto.Server) error {

	headers := []string{
		"server_name",
		"url", "method",
		"interval_sec",
		"timeout_sec",
		"expected_code",
		"status",
	}

	if err := setHeader(xl, "Sheet1", headers); err != nil {
		return fmt.Errorf("failed to set header: %w", err)
	}

	for i, sv := range servers {

		url := ""
		method := "GET"
		interval := 30
		timeout := 10
		expectedCode := 200

		if sv.Endpoint != nil {
			url = sv.Endpoint.URL
			if m, err := utils.ValidateMethod(sv.Endpoint.Method); err == nil {
				method = m
			}
			if sec := int(sv.Endpoint.Interval.Seconds()); sec >= 30 {
				interval = sec
			}
			if sec := int(sv.Endpoint.Timeout.Seconds()); sec >= 10 {
				timeout = sec
			}
			if code := sv.Endpoint.ExpectedCode; code >= 100 && code <= 599 {
				expectedCode = code
			}
		}

		monitorStatus := "offline"
		if sv.MonitorStatus == domain.StatusOn {
			monitorStatus = "online"
		}

		row := i + 2
		values := map[string]string{
			fmt.Sprintf("A%d", row): sv.Name,
			fmt.Sprintf("B%d", row): url,
			fmt.Sprintf("C%d", row): method,
			fmt.Sprintf("D%d", row): fmt.Sprintf("%d", interval),
			fmt.Sprintf("E%d", row): fmt.Sprintf("%d", timeout),
			fmt.Sprintf("F%d", row): fmt.Sprintf("%d", expectedCode),
			fmt.Sprintf("G%d", row): monitorStatus,
		}

		for cell, value := range values {
			if err := xl.SetCellValue("Sheet1", cell, value); err != nil {
				return fmt.Errorf("failed to set cell value: %w", err)
			}
		}
	}

	return nil
}

func (g *ExcelExporter) GenerateExportFile(servers []serverdto.Server) (io.ReadCloser, error) {

	xl := excelize.NewFile()

	if err := fillExportFile(xl, servers); err != nil {
		_ = xl.Close()
		return nil, fmt.Errorf("failed to fill export file: %w", err)
	}

	pr, pw := io.Pipe()
	go func() {

		defer func() { _ = xl.Close() }()

		if err := xl.Write(pw); err != nil {
			err = fmt.Errorf("failed to write Excel file: %w", err)
			_ = pw.CloseWithError(err)
		} else {
			_ = pw.Close()
		}
	}()

	return pr, nil
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
