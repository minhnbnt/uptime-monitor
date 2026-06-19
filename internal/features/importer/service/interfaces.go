package importer

import (
	"io"

	"github.com/minhnbnt/uptime-monitor/internal/features/importer/dto"
)

type ExcelGenerator interface {
	GenerateTemplate(w io.Writer) error
	ParseImportFile(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error)
}
