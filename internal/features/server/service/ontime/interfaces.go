package ontime

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

type OntimeCacheRepository interface {
	MGet(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error)
	MSet(ctx context.Context, items map[dto.BatchGetOntimeItem]float64) error
}
