package dto

import "github.com/minhnbnt/uptime-monitor/internal/domain"

type SearchParams struct {
	Q         string
	Status    *domain.Status
	From      int
	To        int
	SortBy    string
	SortOrder string
}
