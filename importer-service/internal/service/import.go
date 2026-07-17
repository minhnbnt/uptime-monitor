package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/samber/do/v2"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/infrastructure/excel"
)

type ImportService struct {
	serverClient *config.ServerClient
	excelExporter *excel.ExcelExporter
	excelParser   ExcelParser
	logger        *slog.Logger
}

type ExcelParser interface {
	ParseImportFile(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error)
}

func RegisterImportService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ImportService, error) {
		return &ImportService{
			serverClient:  do.MustInvoke[*config.ServerClient](i),
			excelExporter: do.MustInvoke[*excel.ExcelExporter](i),
			excelParser:   do.MustInvoke[*excel.ExcelParser](i),
			logger:        do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (s *ImportService) ImportServers(ctx context.Context, userID uint, file io.Reader) (*dto.ImportResult, error) {

	rows, rowErrors, err := s.excelParser.ParseImportFile(file)
	if err != nil {
		s.logger.Error("failed to parse import file", slog.Any("error", err))
		return nil, fmt.Errorf("%w: %s", apperrors.ErrBadRequest, err.Error())
	}

	if len(rows) == 0 {
		return &dto.ImportResult{RowErrors: rowErrors}, nil
	}

	protoInputs := make([]*serverv1.ServerWithEndpointInput, len(rows))
	for i, r := range rows {
		protoInputs[i] = &serverv1.ServerWithEndpointInput{
			Row:          int32(r.Row),
			Name:         r.Name,
			Url:          r.URL,
			Method:       r.Method,
			ExpectedCode: int32(r.ExpectedCode),
			IntervalMs:   int64(r.Interval) * 1000,
			TimeoutMs:    int64(r.Timeout) * 1000,
			UserId:       uint64(userID),
		}
	}

	resp, err := s.serverClient.BatchCreateServers(ctx, &serverv1.BatchCreateServersRequest{Servers: protoInputs})
	if err != nil {
		s.logger.Error("batch create servers failed", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	var (
		successes   []dto.ImportSuccess
		batchErrors []dto.ImportError
	)

	for _, r := range resp.Results {
		if r.Error == "" {
			successes = append(successes, dto.ImportSuccess{
				Row:      int(r.Row),
				Name:     r.Name,
				URL:      r.Url,
				ServerID: uint(r.ServerId),
			})
		} else {
			batchErrors = append(batchErrors, dto.ImportError{Message: r.Error})
		}
	}

	return &dto.ImportResult{Successes: successes, RowErrors: rowErrors, BatchErrors: batchErrors}, nil
}

var _ ExcelParser = (*excel.ExcelParser)(nil)

func (s *ImportService) GenerateTemplate() (io.ReadCloser, error) {

	reader, err := s.excelExporter.GenerateTemplate()

	if err != nil {
		s.logger.Error("failed to generate template", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	return reader, nil
}

func (s *ImportService) ExportServers(ctx context.Context, userID uint, q string, from, to int, sortBy, sortOrder string) (io.ReadCloser, error) {

	searchResp, err := s.serverClient.SearchServers(ctx, &serverv1.SearchServersRequest{
		UserId:    uint64(userID),
		Q:         q,
		From:      int32(from),
		To:        int32(to),
		SortBy:    sortBy,
		SortOrder: sortOrder,
	})
	if err != nil {
		s.logger.Error("search servers failed", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	servers := protoToExportServers(searchResp.Servers)

	reader, err := s.excelExporter.GenerateExportFile(servers)
	if err != nil {
		s.logger.Error("failed to generate export file", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	return reader, nil
}

func protoToExportServers(in []*serverv1.ServerWithEndpoint) []dto.Server {
	out := make([]dto.Server, len(in))
	for i, p := range in {
		out[i] = dto.Server{
			ID:            uint(p.Id),
			Name:          p.Name,
			MonitorStatus: domain.ServerStatus(p.MonitorStatus),
		}

		if p.Url != "" {
			out[i].Endpoint = &dto.Endpoint{
				URL:           p.Url,
				Method:        p.Method,
				ExpectedCode:  int(p.ExpectedCode),
				MonitorStatus: domain.ServerStatus(p.MonitorStatus),
			}
		}
	}
	return out
}
