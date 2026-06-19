package handler

import (
	"context"
	"io"

	"github.com/minhnbnt/uptime-monitor/internal/features/importer/dto"
)

type ImportService interface {
	ImportServers(ctx context.Context, userID uint, file io.Reader) (*dto.ImportResult, error)
	GenerateTemplate(w io.Writer) error
}
