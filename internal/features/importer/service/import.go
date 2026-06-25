package importer

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
	"github.com/minhnbnt/uptime-monitor/internal/features/importer/dto"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/infrastructure"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	featservice "github.com/minhnbnt/uptime-monitor/internal/features/server/service"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

type ImportService struct {
	serverRepository   featservice.ServerRepository
	endpointRepository featservice.EndpointRepository
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

func buildServers(chunk []dto.ImportRow, userID uint) []domain.Server {
	return lo.Map(chunk, func(r dto.ImportRow, _ int) domain.Server {
		return domain.Server{
			Name:        r.Name,
			CreatedByID: userID,
		}
	})
}

func buildSuccesses(chunk []dto.ImportRow, servers []domain.Server) []dto.ImportSuccess {
	res := make([]dto.ImportSuccess, len(servers))
	for i, sv := range servers {
		res[i] = dto.ImportSuccess{
			Row:      chunk[i].Row,
			Name:     sv.Name,
			URL:      chunk[i].URL,
			ServerID: sv.ID,
		}
	}
	return res
}

func buildEndpoints(chunk []dto.ImportRow, servers []domain.Server) []domain.Endpoint {
	endpoints := make([]domain.Endpoint, 0, len(servers))
	for i, sv := range servers {
		url := chunk[i].URL
		if url == "" {
			continue
		}
		endpoints = append(endpoints, domain.Endpoint{
			ServerID:     sv.ID,
			URL:          url,
			Interval:     time.Duration(chunk[i].Interval) * time.Second,
			Timeout:      time.Duration(chunk[i].Timeout) * time.Second,
			Method:       chunk[i].Method,
			ExpectedCode: chunk[i].ExpectedCode,
		})
	}
	return endpoints
}

func (s *ImportService) ImportServers(ctx context.Context, userID uint, file io.Reader) (*dto.ImportResult, error) {

	rows, rowErrors, err := s.excelGenerator.ParseImportFile(file)
	if err != nil {
		s.logger.Error("failed to parse import file", logger.Error(err))
		return nil, apperrors.ErrBadRequest
	}

	if len(rows) == 0 {
		return &dto.ImportResult{RowErrors: rowErrors}, nil
	}

	var (
		successes   []dto.ImportSuccess
		batchErrors []dto.ImportError
	)

	for chunks := range it.Chunk(slices.Values(rows), chunkSize) {

		servers := buildServers(chunks, userID)

		if err := s.serverRepository.BatchCreateServers(ctx, servers); err != nil {
			s.logger.Error("failed to create servers", logger.Error(err))
			batchErrors = append(batchErrors, dto.ImportError{Message: "failed to create servers"})
			continue
		}

		successes = append(successes, buildSuccesses(chunks, servers)...)

		endpoints := buildEndpoints(chunks, servers)
		if len(endpoints) == 0 {
			continue
		}

		if err := s.endpointRepository.BatchCreateEndpoints(ctx, endpoints); err != nil {
			s.logger.Error("failed to create endpoints", logger.Error(err))
			batchErrors = append(batchErrors, dto.ImportError{Message: "failed to create endpoints"})
		}
	}

	return &dto.ImportResult{Successes: successes, RowErrors: rowErrors, BatchErrors: batchErrors}, nil
}

var _ ExcelGenerator = (*infrastructure.ExcelGenerator)(nil)

func (s *ImportService) GenerateTemplate(w io.Writer) error {
	if err := s.excelGenerator.GenerateTemplate(w); err != nil {
		s.logger.Error("failed to generate template", logger.Error(err))
		return apperrors.ErrInternal
	}
	return nil
}
