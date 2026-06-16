package dto

import "github.com/minhnbnt/uptime-monitor/internal/domain"

type ExportParams struct {
	Q         string
	Status    *domain.Status
	From      int
	To        int
	SortBy    string
	SortOrder string
}
