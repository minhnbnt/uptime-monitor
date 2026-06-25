package handler

import (
	"context"
	"io"

	"github.com/minhnbnt/uptime-monitor/internal/features/importer/dto"
)

type mockImportService struct {
	importServersFn    func(ctx context.Context, userID uint, file io.Reader) (*dto.ImportResult, error)
	generateTemplateFn func(w io.Writer) error
}

func (m *mockImportService) ImportServers(ctx context.Context, userID uint, file io.Reader) (*dto.ImportResult, error) {
	return m.importServersFn(ctx, userID, file)
}

func (m *mockImportService) GenerateTemplate(w io.Writer) error {
	return m.generateTemplateFn(w)
}
