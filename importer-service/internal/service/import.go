package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/infrastructure/excel"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/infrastructure/serverclient"
)

type ImportService struct {
	serverClient  *serverclient.ServerClient
	excelExporter *excel.Exporter
	logger        *slog.Logger
	excelParser   Parser
}

type Parser interface {
	ParseImportFile(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error)
}

func RegisterImportService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ImportService, error) {
		return &ImportService{
			serverClient:  do.MustInvoke[*serverclient.ServerClient](i),
			excelExporter: do.MustInvoke[*excel.Exporter](i),
			excelParser:   do.MustInvoke[*excel.Parser](i),
			logger:        do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (s *ImportService) ImportServers(ctx context.Context, userID uuid.UUID, file io.Reader) (*dto.ImportResult, error) {

	rows, rowErrors, err := s.excelParser.ParseImportFile(file)
	if err != nil {
		s.logger.Error("failed to parse import file", slog.Any("error", err))
		return nil, fmt.Errorf("%w: %s", apperrors.ErrBadRequest, err.Error())
	}

	if len(rows) == 0 {
		return &dto.ImportResult{RowErrors: rowErrors}, nil
	}

	successes, batchErrors, err := s.serverClient.BatchCreateServers(ctx, userID, rows)
	if err != nil {
		s.logger.Error("batch create servers failed", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	return &dto.ImportResult{
		Successes:   successes,
		RowErrors:   rowErrors,
		BatchErrors: batchErrors,
	}, nil
}

var _ Parser = (*excel.Parser)(nil)

func (s *ImportService) GenerateTemplate() (io.ReadCloser, error) {

	reader, err := s.excelExporter.GenerateTemplate()

	if err != nil {
		s.logger.Error("failed to generate template", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	return reader, nil
}

func (s *ImportService) ExportServers(ctx context.Context, userID uuid.UUID, params dto.SearchServersParams) (io.ReadCloser, error) {

	servers, err := s.serverClient.SearchServers(ctx, userID, params)
	if err != nil {
		s.logger.Error("search servers failed", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	reader, err := s.excelExporter.GenerateExportFile(servers)
	if err != nil {
		s.logger.Error("failed to generate export file", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	return reader, nil
}
