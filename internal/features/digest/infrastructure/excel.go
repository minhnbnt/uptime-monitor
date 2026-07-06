package infrastructure

import (
	"fmt"
	"io"
	"iter"
	"maps"
	"slices"
	"time"

	"github.com/samber/lo/it"
	"github.com/xuri/excelize/v2"
)

const (
	ReportSheetName  = "Server Uptime"
	SummarySheetName = "Summary"
)

type ServerRow struct {
	ServerID   uint
	ServerName string
	Stats      map[time.Time]float64
}

type ServerSummary struct {
	Total   int64
	Online  int64
	Offline int64
}

func generateSummarySheet(xl *excelize.File, summary *ServerSummary) error {

	headers := []string{"Metric", "Count"}
	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return fmt.Errorf("failed to create cell name: %w", err)
		}
		if err := xl.SetCellValue(SummarySheetName, cell, h); err != nil {
			return fmt.Errorf("failed to set header: %w", err)
		}
	}

	rows := [][2]string{
		{"Total Servers", fmt.Sprintf("%d", summary.Total)},
		{"Online", fmt.Sprintf("%d", summary.Online)},
		{"Offline", fmt.Sprintf("%d", summary.Offline)},
	}

	for i, row := range rows {

		rowIndex := i + 2

		for colIdx, val := range row {

			cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIndex)
			if err != nil {
				return fmt.Errorf("failed to create cell name: %w", err)
			}

			if err := xl.SetCellValue(SummarySheetName, cell, val); err != nil {
				return fmt.Errorf("failed to set summary value: %w", err)
			}
		}
	}

	return nil
}

func generateReportSheet(xl *excelize.File, rows []ServerRow) error {

	headers := []string{"Server Name"}

	dates := getActiveDate(rows)
	for _, d := range dates {
		headers = append(headers, d.Format("2006-01-02"))
	}

	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return fmt.Errorf("failed to create cell name: %w", err)
		}
		if err := xl.SetCellValue(ReportSheetName, cell, h); err != nil {
			return fmt.Errorf("failed to set header: %w", err)
		}
	}

	for i, row := range rows {

		rowIndex := i + 2

		serverCell, err := excelize.CoordinatesToCellName(1, rowIndex)
		if err != nil {
			return fmt.Errorf("failed to create cell name: %w", err)
		}

		if err := xl.SetCellValue(ReportSheetName, serverCell, row.ServerName); err != nil {
			return fmt.Errorf("failed to set server name: %w", err)
		}

		for colIdx, date := range dates {

			cell, err := excelize.CoordinatesToCellName(colIdx+2, rowIndex)
			if err != nil {
				return fmt.Errorf("failed to create cell name: %w", err)
			}

			pct, ok := row.Stats[date]
			if !ok {

				err = xl.SetCellValue(ReportSheetName, cell, "-")
				if err != nil {
					return fmt.Errorf("failed to set value: %w", err)
				}

				continue
			}

			value := fmt.Sprintf("%.2f%%", pct)
			if err := xl.SetCellValue(ReportSheetName, cell, value); err != nil {
				return fmt.Errorf("failed to set ontime value: %w", err)
			}
		}
	}

	return nil
}

func GenerateStatusReport(rows []ServerRow, summary *ServerSummary) (io.Reader, error) {

	xl := excelize.NewFile()
	defer func() { _ = xl.Close() }()

	if err := xl.SetSheetName("Sheet1", ReportSheetName); err != nil {
		return nil, fmt.Errorf("failed to rename sheet: %w", err)
	}

	if err := generateReportSheet(xl, rows); err != nil {
		return nil, fmt.Errorf("failed to generate report sheet: %w", err)
	}

	if _, err := xl.NewSheet(SummarySheetName); err != nil {
		return nil, fmt.Errorf("failed to create summary sheet: %w", err)
	}

	if err := generateSummarySheet(xl, summary); err != nil {
		return nil, fmt.Errorf("failed to generate summary sheet: %w", err)
	}

	pr, pw := io.Pipe()
	go func() {
		if err := xl.Write(pw); err != nil {
			err = fmt.Errorf("failed to write Excel file: %w", err)
			_ = pw.CloseWithError(err)
		} else {
			pw.Close()
		}
	}()

	return pr, nil
}

func getActiveDate(rows []ServerRow) []time.Time {

	rowsIter := slices.Values(rows)

	activeDates := it.FlatMap(rowsIter, func(row ServerRow) iter.Seq[time.Time] {
		return maps.Keys(row.Stats)
	})

	activeDates = it.Uniq(activeDates)

	return slices.SortedFunc(activeDates, func(a, b time.Time) int {
		return a.Compare(b)
	})
}
