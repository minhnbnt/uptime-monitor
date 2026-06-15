package service

import (
	"context"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/samber/lo/it"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/server/infrastructure"
)

type ExcelGenerator interface {
	GenerateTemplate(w io.Writer) error
	ParseImportFile(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error)
}

type ImportService struct {
	serverRepository   ServerRepository
	endpointRepository EndpointRepository
	excelGenerator     ExcelGenerator
}

func RegisterImportService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ImportService, error) {
		return &ImportService{
			serverRepository:   do.MustInvoke[*serverrepo.ServerRepository](i),
			endpointRepository: do.MustInvoke[*serverrepo.EndpointRepository](i),
			excelGenerator:     do.MustInvoke[*infrastructure.ExcelGenerator](i),
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

	if err := s.serverRepository.BatchCreateServers(ctx, servers); err != nil {
		rowErrors = append(rowErrors, dto.ImportRowError{
			Message: fmt.Sprintf("failed to create servers: %v", err),
		})
		return &dto.ImportResult{Errors: rowErrors}, nil
	}

	serversIter := slices.Values(servers)

	endpoints := it.MapI(serversIter, func(sv domain.Server, index int) domain.Endpoint {
		return domain.Endpoint{
			ServerID:     sv.ID,
			URL:          rows[index].URL,
			Status:       domain.StatusActive,
			Interval:     time.Duration(rows[index].Interval) * time.Second,
			Timeout:      time.Duration(rows[index].Timeout) * time.Second,
			Method:       rows[index].Method,
			ExpectedCode: rows[index].ExpectedCode,
		}
	})

	endpoints = it.Filter(endpoints, func(e domain.Endpoint) bool { return e.URL != "" })

	for chunk := range it.Chunk(endpoints, 100) {
		if err := s.endpointRepository.BatchCreateEndpoints(ctx, chunk); err != nil {
			rowErrors = append(rowErrors, dto.ImportRowError{
				Message: fmt.Sprintf("failed to create endpoints: %v", err),
			})
		}
	}

	return &dto.ImportResult{Imported: len(servers), Errors: rowErrors}, nil
}

func (s *ImportService) GenerateTemplate(w io.Writer) error {
	return s.excelGenerator.GenerateTemplate(w)
}
