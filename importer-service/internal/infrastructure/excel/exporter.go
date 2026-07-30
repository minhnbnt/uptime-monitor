package excel

import (
	"fmt"
	"io"

	"github.com/samber/do/v2"
	"github.com/xuri/excelize/v2"

	serverdto "github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/dto"
)

type Exporter struct{}

func RegisterExporter(i do.Injector) {
	do.Provide(i, func(_ do.Injector) (*Exporter, error) {
		return &Exporter{}, nil
	})
}

func fillTemplate(xl *excelize.File) error {

	headers := []string{
		"server_name",
		"namespace", "kind", "object_id", "container_name",
		"interval_sec",
		"timeout_sec",
	}

	if err := setHeader(xl, "Sheet1", headers); err != nil {
		return fmt.Errorf("failed to set header: %w", err)
	}

	if err := xl.SetCellValue("Sheet1", "A2", "My Server"); err != nil {
		return fmt.Errorf("failed to set cell value: %w", err)
	}
	if err := xl.SetCellValue("Sheet1", "B2", "default"); err != nil {
		return fmt.Errorf("failed to set cell value: %w", err)
	}
	if err := xl.SetCellValue("Sheet1", "C2", "Pod"); err != nil {
		return fmt.Errorf("failed to set cell value: %w", err)
	}
	if err := xl.SetCellValue("Sheet1", "D2", "my-pod"); err != nil {
		return fmt.Errorf("failed to set cell value: %w", err)
	}

	return nil
}

func (g *Exporter) GenerateTemplate() (io.ReadCloser, error) {

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
		"namespace", "kind", "object_id", "container_name",
		"interval_sec",
		"timeout_sec",
	}

	if err := setHeader(xl, "Sheet1", headers); err != nil {
		return fmt.Errorf("failed to set header: %w", err)
	}

	for i, sv := range servers {

		interval := 30
		if sec := int(sv.Interval.Seconds()); sec >= 30 {
			interval = sec
		}

		timeout := 10
		if sec := int(sv.Timeout.Seconds()); sec >= 10 {
			timeout = sec
		}

		row := i + 2
		values := map[string]string{
			fmt.Sprintf("A%d", row): sv.Name,
			fmt.Sprintf("B%d", row): sv.Namespace,
			fmt.Sprintf("C%d", row): sv.Kind,
			fmt.Sprintf("D%d", row): sv.ObjectID,
			fmt.Sprintf("E%d", row): sv.ContainerName,
			fmt.Sprintf("F%d", row): fmt.Sprintf("%d", interval),
			fmt.Sprintf("G%d", row): fmt.Sprintf("%d", timeout),
		}

		for cell, value := range values {
			if err := xl.SetCellValue("Sheet1", cell, value); err != nil {
				return fmt.Errorf("failed to set cell value: %w", err)
			}
		}
	}

	return nil
}

func (g *Exporter) GenerateExportFile(
	servers []serverdto.Server,
) (io.ReadCloser, error) {

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
