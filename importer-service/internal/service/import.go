package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/samber/lo"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/infrastructure/excel"
)

type ImportService struct {
	serverClient  *config.ServerClient
	excelExporter *excel.Exporter
	excelParser   Parser
	logger        *slog.Logger
}

type Parser interface {
	ParseImportFile(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error)
}

func RegisterImportService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ImportService, error) {
		return &ImportService{
			serverClient:  do.MustInvoke[*config.ServerClient](i),
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

	protoInputs := lo.Map(rows, func(r dto.ImportRow, _ int) *serverv1.ServerWithEndpointInput {
		return &serverv1.ServerWithEndpointInput{
			Row:           int32(r.Row),
			Name:          r.Name,
			Namespace:     r.Namespace,
			Kind:          r.Kind,
			ObjectId:      r.ObjectID,
			ContainerName: r.ContainerName,
			IntervalMs:    int64(r.Interval) * 1000,
			TimeoutMs:     int64(r.Timeout) * 1000,
			UserId:        userID.String(),
			HttpConfig:    httpConfigFromRow(r),
		}
	})

	request := serverv1.BatchCreateServersRequest{Servers: protoInputs}
	resp, err := s.serverClient.BatchCreateServers(ctx, &request)
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
				ServerID: uint(r.ServerId),
				Row:      int(r.Row),
				Name:     r.Name,
			})
		} else {
			batchErrors = append(batchErrors, dto.ImportError{Message: r.Error})
		}
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

func (s *ImportService) ExportServers(ctx context.Context, userID uuid.UUID, q string, from, to int, sortBy, sortOrder string) (io.ReadCloser, error) {

	request := serverv1.SearchServersRequest{
		UserId:    userID.String(),
		Q:         q,
		From:      int32(from),
		To:        int32(to),
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	searchResp, err := s.serverClient.SearchServers(ctx, &request)
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
	return lo.Map(in, func(p *serverv1.ServerWithEndpoint, _ int) dto.Server {
		return dto.Server{
			ID:            uint(p.Id),
			Name:          p.Name,
			Namespace:     p.Namespace,
			Kind:          p.Kind,
			ObjectID:      p.ObjectId,
			ContainerName: p.ContainerName,
			HTTPConfig:    dtoHTTPConfig(p.HttpConfig),
		}
	})
}

func dtoHTTPConfig(in *serverv1.HttpConfigInput) *dto.HTTPConfig {

	if in == nil {
		return nil
	}

	return &dto.HTTPConfig{
		Port:          int(in.Port),
		EndpointPath:  in.EndpointPath,
		ExpectedCode:  int(in.ExpectedCode),
		BodyCheckExpr: in.BodyCheckExpr,
		Method:        in.Method,
	}
}

func httpConfigFromRow(r dto.ImportRow) *serverv1.HttpConfigInput {

	if r.HTTPPort == 0 &&
		r.HTTPPath == "" &&
		r.HTTPMethod == "" &&

		r.HTTPBodyCheck == "" &&
		r.HTTPExpectedCode == 0 {
		return nil
	}

	return &serverv1.HttpConfigInput{
		Port:          int32(r.HTTPPort),
		EndpointPath:  r.HTTPPath,
		ExpectedCode:  int32(r.HTTPExpectedCode),
		BodyCheckExpr: r.HTTPBodyCheck,
		Method:        r.HTTPMethod,
	}
}
