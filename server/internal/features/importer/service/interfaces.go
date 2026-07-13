package importer

import (
	"io"

	"github.com/minhnbnt/uptime-monitor/internal/features/importer/dto"
)

type ExcelParser interface {
	ParseImportFile(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error)
}
