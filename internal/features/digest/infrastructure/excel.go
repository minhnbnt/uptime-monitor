package infrastructure

import (
	"fmt"
	"io"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type ReportRow struct {
	ServerName string
	URL        string
	Status     domain.ServerStatus
	Time       time.Time
}

func GenerateStatusReport(rows []ReportRow) (io.Reader, error) {

	xl := excelize.NewFile()
	defer xl.Close()

	headers := []string{"Server Name", "URL", "Status", "Time"}
	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to create cell name: %w", err)
		}
		if err := xl.SetCellValue("Sheet1", cell, h); err != nil {
			return nil, fmt.Errorf("failed to set header: %w", err)
		}
	}

	for i, row := range rows {

		r := i + 2

		values := map[string]string{
			fmt.Sprintf("A%d", r): row.ServerName,
			fmt.Sprintf("B%d", r): row.URL,
			fmt.Sprintf("C%d", r): string(row.Status),
			fmt.Sprintf("D%d", r): row.Time.Format("2006-01-02 15:04:05"),
		}

		for cell, value := range values {
			if err := xl.SetCellValue("Sheet1", cell, value); err != nil {
				return nil, fmt.Errorf("failed to set cell value: %w", err)
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
