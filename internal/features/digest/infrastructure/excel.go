package infrastructure

import (
	"fmt"
	"io"
	"time"

	"github.com/xuri/excelize/v2"
)

type ServerRow struct {
	ServerID   uint
	ServerName string
	Stats      map[time.Time]float64
}

func GenerateStatusReport(rows []ServerRow, dates []time.Time) (io.Reader, error) {

	xl := excelize.NewFile()
	defer func() { _ = xl.Close() }()

	headers := []string{"Server Name"}
	for _, d := range dates {
		headers = append(headers, d.Format("2006-01-02"))
	}

	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to create cell name: %w", err)
		}
		if err := xl.SetCellValue("Sheet1", cell, h); err != nil {
			return nil, fmt.Errorf("failed to set header: %w", err)
		}
	}

	for rowIdx, row := range rows {

		r := rowIdx + 2

		serverCell, _ := excelize.CoordinatesToCellName(1, r)
		if err := xl.SetCellValue("Sheet1", serverCell, row.ServerName); err != nil {
			return nil, fmt.Errorf("failed to set server name: %w", err)
		}

		for colIdx, date := range dates {

			cell, _ := excelize.CoordinatesToCellName(colIdx+2, r)

			pct, ok := row.Stats[date]
			if !ok {
				_ = xl.SetCellValue("Sheet1", cell, "-")
				continue
			}

			value := fmt.Sprintf("%.2f%%", pct)
			if err := xl.SetCellValue("Sheet1", cell, value); err != nil {
				return nil, fmt.Errorf("failed to set ontime value: %w", err)
			}
		}
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
