package importer

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/importer/dto"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

func TestImportService_ImportServers(t *testing.T) {
	const userID uint = 1

	t.Run("success", func(t *testing.T) {
		rows := []dto.ImportRow{
			{Name: "server-a", URL: "https://a.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
			{Name: "server-b", URL: "https://b.org/ping", Method: "POST", Interval: 60, Timeout: 15, ExpectedCode: 201},
		}
		var gotServers []domain.Server
		var gotEndpoints []domain.Endpoint

		svc := &ImportService{
			serverRepository: &mockServerRepo{
				batchCreateServersFn: func(_ context.Context, servers []domain.Server) error {
					for i := range servers {
						servers[i].ID = uint(i + 1)
					}
					gotServers = servers
					return nil
				},
			},
			endpointRepository: &mockEndpointRepo{
				batchCreateEndpointsFn: func(_ context.Context, eps []domain.Endpoint) error {
					gotEndpoints = eps
					return nil
				},
			},
			excelGenerator: &mockExcelGenerator{
				parseImportFileFn: func(_ io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error) {
					return rows, nil, nil
				},
			},
		}

		result, err := svc.ImportServers(t.Context(), userID, bytes.NewReader(nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Successes) != 2 {
			t.Errorf("len(Successes) = %d, want 2", len(result.Successes))
		}
		if len(result.RowErrors)+len(result.BatchErrors) != 0 {
			t.Errorf("unexpected errors: row=%v batch=%v", result.RowErrors, result.BatchErrors)
		}
		if len(gotServers) != 2 {
			t.Fatalf("got %d servers, want 2", len(gotServers))
		}
		if gotServers[0].Name != "server-a" || gotServers[1].Name != "server-b" {
			t.Errorf("server names = %v", gotServers)
		}
		if gotServers[0].CreatedByID != userID || gotServers[1].CreatedByID != userID {
			t.Errorf("CreatedByID not set: %+v", gotServers)
		}
		if gotServers[0].ID != 1 || gotServers[1].ID != 2 {
			t.Errorf("server IDs not populated: %+v", gotServers)
		}
		if len(gotEndpoints) != 2 {
			t.Fatalf("got %d endpoints, want 2", len(gotEndpoints))
		}
		if gotEndpoints[0].ServerID != 1 || gotEndpoints[1].ServerID != 2 {
			t.Errorf("endpoint ServerID mismatch: %+v", gotEndpoints)
		}
		if gotEndpoints[0].URL != "https://a.com" || gotEndpoints[1].URL != "https://b.org/ping" {
			t.Errorf("endpoint URLs = %v", gotEndpoints)
		}
	})

	t.Run("skip empty URL", func(t *testing.T) {
		rows := []dto.ImportRow{
			{Name: "server-a", URL: "https://a.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
			{Name: "server-b", URL: "", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		}
		var gotEndpoints []domain.Endpoint

		svc := &ImportService{
			serverRepository: &mockServerRepo{
				batchCreateServersFn: func(_ context.Context, servers []domain.Server) error {
					for i := range servers {
						servers[i].ID = uint(i + 1)
					}
					return nil
				},
			},
			endpointRepository: &mockEndpointRepo{
				batchCreateEndpointsFn: func(_ context.Context, eps []domain.Endpoint) error {
					gotEndpoints = eps
					return nil
				},
			},
			excelGenerator: &mockExcelGenerator{
				parseImportFileFn: func(_ io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error) {
					return rows, nil, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		result, err := svc.ImportServers(t.Context(), userID, bytes.NewReader(nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Successes) != 2 {
			t.Errorf("len(Successes) = %d, want 2", len(result.Successes))
		}
		if len(gotEndpoints) != 1 {
			t.Fatalf("got %d endpoints, want 1", len(gotEndpoints))
		}
		if gotEndpoints[0].ServerID != 1 {
			t.Errorf("ServerID = %d, want 1", gotEndpoints[0].ServerID)
		}
	})

	t.Run("parse error", func(t *testing.T) {
		svc := &ImportService{
			excelGenerator: &mockExcelGenerator{
				parseImportFileFn: func(_ io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error) {
					return nil, nil, errors.New("invalid excel file")
				},
			},
			logger: logger.NewMockLogger(),
		}

		_, err := svc.ImportServers(t.Context(), userID, bytes.NewReader(nil))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty rows", func(t *testing.T) {
		svc := &ImportService{
			excelGenerator: &mockExcelGenerator{
				parseImportFileFn: func(_ io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error) {
					return nil, nil, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		result, err := svc.ImportServers(t.Context(), userID, bytes.NewReader(nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Successes) != 0 {
			t.Errorf("len(Successes) = %d, want 0", len(result.Successes))
		}
	})

	t.Run("parse row errors", func(t *testing.T) {
		rowErrs := []dto.ImportRowError{
			{Row: 2, Message: "invalid name"},
			{Row: 3, Message: "invalid url"},
		}

		svc := &ImportService{
			excelGenerator: &mockExcelGenerator{
				parseImportFileFn: func(_ io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error) {
					return nil, rowErrs, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		result, err := svc.ImportServers(t.Context(), userID, bytes.NewReader(nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Successes) != 0 {
			t.Errorf("len(Successes) = %d, want 0", len(result.Successes))
		}
		if len(result.RowErrors) != 2 {
			t.Fatalf("got %d row errors, want 2", len(result.RowErrors))
		}
	})

	t.Run("parse row errors with partial valid rows", func(t *testing.T) {
		rows := []dto.ImportRow{
			{Name: "server-a", URL: "https://a.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		}
		rowErrs := []dto.ImportRowError{
			{Row: 2, Message: "invalid name"},
		}

		svc := &ImportService{
			serverRepository: &mockServerRepo{
				batchCreateServersFn: func(_ context.Context, servers []domain.Server) error {
					servers[0].ID = 1
					return nil
				},
			},
			endpointRepository: &mockEndpointRepo{
				batchCreateEndpointsFn: func(_ context.Context, _ []domain.Endpoint) error {
					return nil
				},
			},
			excelGenerator: &mockExcelGenerator{
				parseImportFileFn: func(_ io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error) {
					return rows, rowErrs, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		result, err := svc.ImportServers(t.Context(), userID, bytes.NewReader(nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Successes) != 1 {
			t.Errorf("len(Successes) = %d, want 1", len(result.Successes))
		}
		if len(result.RowErrors) != 1 {
			t.Fatalf("got %d row errors, want 1", len(result.RowErrors))
		}
	})

	t.Run("server batch create error", func(t *testing.T) {
		rows := []dto.ImportRow{
			{Name: "server-a", URL: "https://a.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		}

		svc := &ImportService{
			serverRepository: &mockServerRepo{
				batchCreateServersFn: func(_ context.Context, _ []domain.Server) error {
					return errors.New("connection refused")
				},
			},
			excelGenerator: &mockExcelGenerator{
				parseImportFileFn: func(_ io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error) {
					return rows, nil, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		result, err := svc.ImportServers(t.Context(), userID, bytes.NewReader(nil))
		if err != nil {
			t.Fatalf("did not expect error from service: %v", err)
		}
		if len(result.Successes) != 0 {
			t.Errorf("len(Successes) = %d, want 0", len(result.Successes))
		}
		if len(result.BatchErrors) != 1 {
			t.Fatalf("got %d batch errors, want 1", len(result.BatchErrors))
		}
		if result.BatchErrors[0].Message != "failed to create servers" {
			t.Errorf("Message = %q, want %q", result.BatchErrors[0].Message, "failed to create servers")
		}
	})

	t.Run("endpoint batch create error", func(t *testing.T) {
		rows := []dto.ImportRow{
			{Name: "server-a", URL: "https://a.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
			{Name: "server-b", URL: "https://b.com", Method: "GET", Interval: 30, Timeout: 10, ExpectedCode: 200},
		}

		var createdEndpoints int
		svc := &ImportService{
			serverRepository: &mockServerRepo{
				batchCreateServersFn: func(_ context.Context, servers []domain.Server) error {
					for i := range servers {
						servers[i].ID = uint(i + 1)
					}
					return nil
				},
			},
			endpointRepository: &mockEndpointRepo{
				batchCreateEndpointsFn: func(_ context.Context, _ []domain.Endpoint) error {
					createdEndpoints++
					return errors.New("timeout")
				},
			},
			excelGenerator: &mockExcelGenerator{
				parseImportFileFn: func(_ io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error) {
					return rows, nil, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		result, err := svc.ImportServers(t.Context(), userID, bytes.NewReader(nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Successes) != 2 {
			t.Errorf("len(Successes) = %d, want 2", len(result.Successes))
		}
		if len(result.BatchErrors) != 1 {
			t.Fatalf("got %d batch errors, want 1", len(result.BatchErrors))
		}
		if result.BatchErrors[0].Message != "failed to create endpoints" {
			t.Errorf("Message = %q", result.BatchErrors[0].Message)
		}
	})
}

func TestImportService_GenerateTemplate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var buf bytes.Buffer
		svc := &ImportService{
			excelGenerator: &mockExcelGenerator{
				generateTemplateFn: func(w io.Writer) error {
					_, err := w.Write([]byte("template data"))
					return err
				},
			},
			logger: logger.NewMockLogger(),
		}

		err := svc.GenerateTemplate(&buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.String() != "template data" {
			t.Errorf("got %q, want %q", buf.String(), "template data")
		}
	})

	t.Run("generator error", func(t *testing.T) {
		svc := &ImportService{
			excelGenerator: &mockExcelGenerator{
				generateTemplateFn: func(_ io.Writer) error {
					return errors.New("template error")
				},
			},
			logger: logger.NewMockLogger(),
		}

		err := svc.GenerateTemplate(bytes.NewBuffer(nil))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
