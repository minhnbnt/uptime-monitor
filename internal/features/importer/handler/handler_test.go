package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/features/importer/dto"
)

func TestImportHandler_ImportServers(t *testing.T) {
	t.Run("success with results", func(t *testing.T) {
		h := &ImportHandler{
			importService: &mockImportService{
				importServersFn: func(_ context.Context, _ uint, _ io.Reader) (*dto.ImportResult, error) {
					return &dto.ImportResult{
						Successes: []dto.ImportSuccess{
							{Row: 1, Name: "s1", URL: "https://a.com", ServerID: 10},
							{Row: 2, Name: "s2", URL: "https://b.com", ServerID: 20},
						},
						RowErrors: []dto.ImportRowError{
							{Row: 3, Message: "invalid name"},
						},
					}, nil
				},
			},
		}

		resp, err := h.ImportServers(context.Background(), &api.ImportServersReq{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.SuccessCount != 2 {
			t.Errorf("SuccessCount = %d, want 2", resp.SuccessCount)
		}
		if len(resp.Successes) != 2 {
			t.Errorf("len(Successes) = %d, want 2", len(resp.Successes))
		}
		if resp.FailedCount != 1 {
			t.Errorf("FailedCount = %d, want 1", resp.FailedCount)
		}
		if len(resp.Failed) != 1 {
			t.Errorf("len(Failed) = %d, want 1", len(resp.Failed))
		}
		if resp.Successes[0].ServerID.Value != 10 {
			t.Errorf("ServerID = %d", resp.Successes[0].ServerID.Value)
		}
	})

	t.Run("success with batch errors", func(t *testing.T) {
		h := &ImportHandler{
			importService: &mockImportService{
				importServersFn: func(_ context.Context, _ uint, _ io.Reader) (*dto.ImportResult, error) {
					return &dto.ImportResult{
						BatchErrors: []dto.ImportError{
							{Message: "batch error"},
						},
					}, nil
				},
			},
		}

		resp, err := h.ImportServers(context.Background(), &api.ImportServersReq{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.FailedCount != 1 {
			t.Errorf("FailedCount = %d, want 1", resp.FailedCount)
		}
	})

	t.Run("empty result", func(t *testing.T) {
		h := &ImportHandler{
			importService: &mockImportService{
				importServersFn: func(_ context.Context, _ uint, _ io.Reader) (*dto.ImportResult, error) {
					return &dto.ImportResult{}, nil
				},
			},
		}

		resp, err := h.ImportServers(context.Background(), &api.ImportServersReq{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.SuccessCount != 0 || resp.FailedCount != 0 {
			t.Errorf("expected zero counts, got %d/%d", resp.SuccessCount, resp.FailedCount)
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := &ImportHandler{
			importService: &mockImportService{
				importServersFn: func(_ context.Context, _ uint, _ io.Reader) (*dto.ImportResult, error) {
					return nil, errors.New("import failed")
				},
			},
		}

		_, err := h.ImportServers(context.Background(), &api.ImportServersReq{})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d", statusErr.StatusCode)
		}
	})
}

func TestImportHandler_DownloadImportTemplate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := &ImportHandler{
			importService: &mockImportService{
				generateTemplateFn: func(w io.Writer) error {
					_, _ = w.Write([]byte("template content"))
					return nil
				},
			},
		}

		resp, err := h.DownloadImportTemplate(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, readErr := io.ReadAll(resp.Data)
		if readErr != nil {
			t.Fatalf("read data: %v", readErr)
		}
		if string(data) != "template content" {
			t.Errorf("data = %q", string(data))
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := &ImportHandler{
			importService: &mockImportService{
				generateTemplateFn: func(_ io.Writer) error {
					return errors.New("template error")
				},
			},
		}

		_, err := h.DownloadImportTemplate(context.Background())
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d", statusErr.StatusCode)
		}
	})
}
