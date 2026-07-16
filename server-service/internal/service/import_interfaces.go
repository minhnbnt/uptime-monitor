package service

import (
	"io"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
)

type ExcelParser interface {
	ParseImportFile(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error)
}
