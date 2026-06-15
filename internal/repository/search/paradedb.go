package search

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type ParadeDBSearcher struct {
	db *gorm.DB
}

func RegisterParadeDBSearcher(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ParadeDBSearcher, error) {

		gw := do.MustInvoke[*config.GORMWrapper](i)
		if !gw.SearchEnabled {
			return nil, errors.New("pg_search not available, ParadeDB search disabled")
		}

		return &ParadeDBSearcher{db: gw.GetDB()}, nil
	})
}

func (s *ParadeDBSearcher) Search(
	ctx context.Context, q string, createdByID uint, limit, offset int,
) ([]domain.Server, int64, error) {

	total, err := gorm.G[domain.Server](s.db).
		Where("created_by_id = ? AND name @@@ ?", createdByID, q).
		Count(ctx, "*")

	if err != nil {
		return nil, 0, fmt.Errorf("search count: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	servers, err := gorm.G[domain.Server](s.db).
		Where("created_by_id = ? AND name @@@ ?", createdByID, q).
		Order("pdb.score(id) DESC").
		Find(ctx)

	if err != nil {
		return nil, 0, fmt.Errorf("search query: %w", err)
	}

	return servers, total, nil
}
