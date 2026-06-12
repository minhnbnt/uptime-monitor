package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/server/infrastructure"
)

type ExcelGenerator interface {
	GenerateTemplate(w io.Writer) error
	ParseImportFile(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error)
}

type ImportService struct {
	db             *gorm.DB
	excelGenerator ExcelGenerator
}

func RegisterImportService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ImportService, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return &ImportService{
			db:             dbWrapper.GetDB(),
			excelGenerator: do.MustInvoke[*infrastructure.ExcelGenerator](i),
		}, nil
	})
}

func (s *ImportService) ImportServers(ctx context.Context, userID uint, file io.Reader) (*dto.ImportResult, error) {
	rows, rowErrors, err := s.excelGenerator.ParseImportFile(file)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return &dto.ImportResult{Errors: rowErrors}, nil
	}

	servers := lo.Map(rows, func(r dto.ImportRow, _ int) domain.Server {
		return domain.Server{
			Name:        r.Name,
			Status:      domain.StatusActive,
			CreatedByID: userID,
		}
	})

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		for _, chunk := range lo.Chunk(servers, 100) {
			if err := tx.Create(&chunk).Error; err != nil {
				return fmt.Errorf("failed to batch create servers: %w", err)
			}
		}

		var endpoints []domain.Endpoint
		for i, s := range servers {
			if rows[i].URL == "" {
				continue
			}
			endpoints = append(endpoints, domain.Endpoint{
				ServerID:     s.ID,
				URL:          rows[i].URL,
				Status:       domain.StatusActive,
				Interval:     time.Duration(rows[i].Interval) * time.Second,
				Timeout:      time.Duration(rows[i].Timeout) * time.Second,
				Method:       rows[i].Method,
				ExpectedCode: rows[i].ExpectedCode,
			})
		}

		for _, chunk := range lo.Chunk(endpoints, 100) {
			if err := tx.Create(&chunk).Error; err != nil {
				return fmt.Errorf("failed to batch create endpoints: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &dto.ImportResult{
		Imported: len(rows),
		Errors:   rowErrors,
	}, nil
}

func (s *ImportService) GenerateTemplate(w io.Writer) error {
	return s.excelGenerator.GenerateTemplate(w)
}
