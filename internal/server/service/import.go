package service

import (
	"context"
	"fmt"
	"io"

	"github.com/samber/do/v2"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

type ImportService struct {
	db *gorm.DB
}

func RegisterImportService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ImportService, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return &ImportService{db: dbWrapper.GetDB()}, nil
	})
}

func (s *ImportService) ImportServers(ctx context.Context, userID uint, file io.Reader) (*dto.ImportResult, error) {
	return &dto.ImportResult{}, fmt.Errorf("import not yet implemented")
}

func (s *ImportService) GenerateTemplate(w io.Writer) error {
	xl := excelize.NewFile()
	defer xl.Close()

	headers := []string{"server_name", "url", "method", "interval_sec", "timeout_sec", "expected_code"}
	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return fmt.Errorf("failed to create cell name: %w", err)
		}
		if err := xl.SetCellValue("Sheet1", cell, h); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
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
