package search

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

var sortFieldMap = map[string]string{
	"name":       "name",
	"created_at": "created_at",
	"status":     "status",
	"score":      "pdb.score(id)",
}

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
	ctx context.Context, params dto.SearchParams, createdByID uint,
) ([]domain.Server, int64, error) {

	field, ok := sortFieldMap[params.SortBy]
	if !ok {
		field = sortFieldMap["score"]
	}
	order := "DESC"
	if strings.EqualFold(params.SortOrder, "asc") {
		order = "ASC"
	}
	safeOrder := fmt.Sprintf("%s %s", field, order)

	limit := params.PerPage
	offset := (params.Page - 1) * params.PerPage

	total, err := gorm.G[domain.Server](s.db).
		Where("created_by_id = ? AND name @@@ ?", createdByID, params.Q).
		Count(ctx, "*")

	if err != nil {
		return nil, 0, fmt.Errorf("search count: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	servers, err := gorm.G[domain.Server](s.db).
		Where("created_by_id = ? AND name @@@ ?", createdByID, params.Q).
		Order(safeOrder).
		Limit(limit).
		Offset(offset).
		Find(ctx)

	if err != nil {
		return nil, 0, fmt.Errorf("search query: %w", err)
	}

	return servers, total, nil
}
