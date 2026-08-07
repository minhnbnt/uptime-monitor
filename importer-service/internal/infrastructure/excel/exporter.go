package excel

import (
	"fmt"
	"io"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/xuri/excelize/v2"

	serverdto "github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/dto"
)

type Exporter struct{}

const sheetName = "Servers"

func RegisterExporter(i do.Injector) {
	do.Provide(i, func(_ do.Injector) (*Exporter, error) {
		return &Exporter{}, nil
	})
}

func (g *Exporter) GenerateTemplate() (io.ReadCloser, error) {

	examples := []serverdto.Server{{

		Name:      "My Server",
		Namespace: "default",
		Kind:      "Pod",
		ObjectID:  "my-pod",
		Interval:  30 * time.Second,
		Timeout:   10 * time.Second,

		HTTPConfig: &serverdto.HTTPConfig{
			Port:         8080,
			EndpointPath: "/healthz",
			ExpectedCode: 200,
			Method:       "GET",
		},
	}}

	return g.GenerateExportFile(examples)
}

func fillServers(xl *excelize.File, servers []serverdto.Server) error {

	if err := setHeader(xl, sheetName, headers); err != nil {
		return fmt.Errorf("failed to set header: %w", err)
	}

	for i, sv := range servers {
		for cell, value := range toRowMap(i+2, sv) {
			if err := xl.SetCellValue(sheetName, cell, value); err != nil {
				return fmt.Errorf("failed to set cell value: %w", err)
			}
		}
	}

	return nil
}

func (g *Exporter) fillExportFile(xl *excelize.File, servers []serverdto.Server) error {

	if err := xl.SetSheetName("Sheet1", sheetName); err != nil {
		return fmt.Errorf("failed to rename sheet: %w", err)
	}

	if err := fillServers(xl, servers); err != nil {
		return fmt.Errorf("failed to fill export file: %w", err)
	}

	return nil
}

func (g *Exporter) GenerateExportFile(servers []serverdto.Server) (io.ReadCloser, error) {

	pr, pw := io.Pipe()
	go func() {

		xl := excelize.NewFile()
		defer func() { _ = xl.Close() }()

		if err := g.fillExportFile(xl, servers); err != nil {
			_ = pw.CloseWithError(err)
			return
		}

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

func cellRef(header string, rowIndex int) string {

	cell, err := excelize.CoordinatesToCellName(headerColumns[header], rowIndex)
	if err != nil {
		return ""
	}

	return cell
}

func toRowMap(rowIndex int, sv serverdto.Server) map[string]string {

	cellMap := lo.SliceToMap(headers, func(label string) (string, string) {
		return label, cellRef(label, rowIndex)
	})

	values := map[string]string{
		cellMap["server_name"]:    sv.Name,
		cellMap["namespace"]:      sv.Namespace,
		cellMap["kind"]:           sv.Kind,
		cellMap["object_id"]:      sv.ObjectID,
		cellMap["container_name"]: sv.ContainerName,
	}

	interval := int(sv.Interval.Seconds())
	if interval >= 1 {
		values[cellMap["interval_sec"]] = fmt.Sprintf("%d", interval)
	}

	timeout := int(sv.Timeout.Seconds())
	if timeout >= 1 {
		values[cellMap["timeout_sec"]] = fmt.Sprintf("%d", timeout)
	}

	if sv.HTTPConfig == nil {
		return values
	}

	if sv.HTTPConfig.Port > 0 {
		values[cellMap["http_port"]] = fmt.Sprintf("%d", sv.HTTPConfig.Port)
	}
	if sv.HTTPConfig.EndpointPath != "" {
		values[cellMap["http_path"]] = sv.HTTPConfig.EndpointPath
	}
	if sv.HTTPConfig.ExpectedCode > 0 {
		values[cellMap["http_expected_code"]] = fmt.Sprintf("%d", sv.HTTPConfig.ExpectedCode)
	}
	if sv.HTTPConfig.BodyCheckExpr != "" {
		values[cellMap["http_body_check"]] = sv.HTTPConfig.BodyCheckExpr
	}
	if sv.HTTPConfig.Method != "" {
		values[cellMap["http_method"]] = sv.HTTPConfig.Method
	}

	return values
}
