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

const chunkSize = 100

func (s *ImportService) ImportServers(ctx context.Context, userID uint, file io.Reader) (*dto.ImportResult, error) {

	rows, rowErrors, err := s.excelGenerator.ParseImportFile(file)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return &dto.ImportResult{RowErrors: rowErrors}, nil
	}

	rowIter := slices.Values(rows)

	var (
		successes   []dto.ImportSuccess
		batchErrors []dto.ImportError
	)

	for chunks := range it.Chunk(rowIter, chunkSize) {

		servers := lo.Map(chunks, func(r dto.ImportRow, _ int) domain.Server {
			return domain.Server{
				Name:        r.Name,
				Status:      domain.StatusActive,
				CreatedByID: userID,
			}
		})

		err := s.serverRepository.BatchCreateServers(ctx, servers)
		if err != nil {
			batchErrors = append(batchErrors, dto.ImportError{
				Message: fmt.Sprintf("failed to create servers: %v", err),
			})
			continue
		}

		for i, sv := range servers {
			successes = append(successes, dto.ImportSuccess{
				Row:      chunks[i].Row,
				Name:     sv.Name,
				URL:      chunks[i].URL,
				ServerID: sv.ID,
			})
		}

		serverIter := slices.Values(servers)
		endpoints := it.MapI(serverIter, func(sv domain.Server, index int) domain.Endpoint {
			return domain.Endpoint{
				ServerID:     sv.ID,
				URL:          chunks[index].URL,
				Status:       domain.StatusActive,
				Interval:     time.Duration(chunks[index].Interval) * time.Second,
				Timeout:      time.Duration(chunks[index].Timeout) * time.Second,
				Method:       chunks[index].Method,
				ExpectedCode: chunks[index].ExpectedCode,
			}
		})

		endpoints = it.Filter(endpoints, func(e domain.Endpoint) bool { return e.URL != "" })

		if err := s.endpointRepository.BatchCreateEndpoints(ctx, slices.Collect(endpoints)); err != nil {
			batchErrors = append(batchErrors, dto.ImportError{
				Message: fmt.Sprintf("failed to create endpoints: %v", err),
			})
		}
	}

	return &dto.ImportResult{Successes: successes, RowErrors: rowErrors, BatchErrors: batchErrors}, nil
}

func (s *ImportService) GenerateTemplate(w io.Writer) error {
	return s.excelGenerator.GenerateTemplate(w)
}
