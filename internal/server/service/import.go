package service

import (
	"context"
	"io"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/samber/lo/it"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
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
	logger             logger.Logger
}

func RegisterImportService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ImportService, error) {
		return &ImportService{
			serverRepository:   do.MustInvoke[*serverrepo.ServerRepository](i),
			endpointRepository: do.MustInvoke[*serverrepo.EndpointRepository](i),
			excelGenerator:     do.MustInvoke[*infrastructure.ExcelGenerator](i),
			logger:             do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

const chunkSize = 100

func (s *ImportService) ImportServers(ctx context.Context, userID uint, file io.Reader) (*dto.ImportResult, error) {

	rows, rowErrors, err := s.excelGenerator.ParseImportFile(file)
	if err != nil {
		s.logger.Error("failed to parse import file", logger.Error(err))
		return nil, apperrors.ErrBadRequest
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
			s.logger.Error("failed to create servers", logger.Error(err))
			batchErrors = append(batchErrors, dto.ImportError{
				Message: "failed to create servers",
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
			s.logger.Error("failed to create endpoints", logger.Error(err))
			batchErrors = append(batchErrors, dto.ImportError{
				Message: "failed to create endpoints",
			})
		}
	}

	return &dto.ImportResult{Successes: successes, RowErrors: rowErrors, BatchErrors: batchErrors}, nil
}

func (s *ImportService) GenerateTemplate(w io.Writer) error {
	if err := s.excelGenerator.GenerateTemplate(w); err != nil {
		s.logger.Error("failed to generate template", logger.Error(err))
		return apperrors.ErrInternal
	}
	return nil
}
