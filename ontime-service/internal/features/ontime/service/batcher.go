package ontime

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/samber/lo/it"

	"github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/features/ontime/repository"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

func NewBatcher(repo OntineRepository, cache *ontimerepo.OntimeCacheRepository, l *slog.Logger) *Batcher {
	var cacheInterface OntimeCacheRepository
	if cache != nil {
		cacheInterface = cache
	}
	return &Batcher{
		ontineRepository:      repo,
		ontimeCacheRepository: cacheInterface,
		logger:                l,
		calculator:            OntimeCalculator{},
	}
}

func RegisterBatcher(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Batcher, error) {
		return NewBatcher(
			do.MustInvoke[*ontimerepo.OntineRepository](i),
			do.MustInvoke[*ontimerepo.OntimeCacheRepository](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

type OntineRepository interface {
	BatchGetOntime(ctx context.Context, req []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.RawEvent, error)
}

type Batcher struct {
	ontineRepository      OntineRepository
	ontimeCacheRepository OntimeCacheRepository
	logger                *slog.Logger
	calculator            OntimeCalculator
}

func (b *Batcher) BatchGetOntimeUntil(ctx context.Context, req []dto.BatchGetOntimeItem, until time.Time) ([]dto.BatchGetOntimeResponse, error) {

	cacheKeys := getCacheKey(req)
	resultMap := b.resolveCache(ctx, cacheKeys)

	missKeys := lo.Filter(cacheKeys, func(key dto.BatchGetOntimeItem, _ int) bool {
		_, hit := resultMap[key]
		return !hit
	})

	if len(missKeys) == 0 {
		return b.buildResponse(req, resultMap), nil
	}

	toCache := b.fillMisses(ctx, missKeys, until)
	maps.Copy(resultMap, toCache)

	if b.ontimeCacheRepository != nil {
		if err := b.ontimeCacheRepository.MSet(ctx, toCache); err != nil {
			b.logger.Warn("failed to batch cache ontime results", slog.Any("error", err))
		}
	}

	return b.buildResponse(req, resultMap), nil
}

func (b *Batcher) BatchGetOntime(ctx context.Context, req []dto.BatchGetOntimeItem) ([]dto.BatchGetOntimeResponse, error) {
	return b.BatchGetOntimeUntil(ctx, req, time.Now())
}

func getCacheKey(req []dto.BatchGetOntimeItem) []dto.BatchGetOntimeItem {

	reqIter := slices.Values(req)

	cacheKeys := it.Map(reqIter, func(item dto.BatchGetOntimeItem) dto.BatchGetOntimeItem {
		item.Date = utils.TruncateDay(item.Date)
		return item
	})

	cacheKeys = it.Uniq(cacheKeys)

	return slices.Collect(cacheKeys)
}

func (b *Batcher) resolveCache(ctx context.Context, keys []dto.BatchGetOntimeItem) map[dto.BatchGetOntimeItem]float64 {

	if b.ontimeCacheRepository == nil {
		return make(map[dto.BatchGetOntimeItem]float64, len(keys))
	}

	cached, err := b.ontimeCacheRepository.MGet(ctx, keys)

	if err != nil {
		b.logger.Warn("ontime cache MGet failed, falling back to DB", slog.Any("error", err))
		return make(map[dto.BatchGetOntimeItem]float64, len(keys))
	}

	return cached
}

func (b *Batcher) fillMisses(ctx context.Context, missedKeys []dto.BatchGetOntimeItem, until time.Time) map[dto.BatchGetOntimeItem]float64 {

	requests := lo.Map(missedKeys, func(key dto.BatchGetOntimeItem, _ int) ontimerepo.BatchGetOntimeRequest {
		return ontimerepo.BatchGetOntimeRequest{ServerID: key.ServerID, Date: key.Date}
	})

	rows, err := b.ontineRepository.BatchGetOntime(ctx, requests)
	if err != nil {
		b.logger.Warn("failed to get missed ontime keys", slog.Any("error", err))
		return make(map[dto.BatchGetOntimeItem]float64)
	}

	groups := lo.GroupBy(rows, func(row ontimerepo.RawEvent) serverDayKey {
		return serverDayKey{ServerID: row.ServerID, Day: row.Day}
	})

	dayUntil := utils.TruncateDay(until)
	toCache := lo.SliceToMap(missedKeys, func(key dto.BatchGetOntimeItem) (dto.BatchGetOntimeItem, float64) {
		events := groups[serverDayKey{ServerID: key.ServerID, Day: key.Date}]
		return key, b.calculator.CalculateDayOntime(events, dayUntil, until)
	})

	return toCache
}

func (b *Batcher) buildResponse(req []dto.BatchGetOntimeItem, resultMap map[dto.BatchGetOntimeItem]float64) []dto.BatchGetOntimeResponse {

	groups := lo.GroupBy(req, func(item dto.BatchGetOntimeItem) uint {
		return item.ServerID
	})

	return lo.MapToSlice(groups, func(serverID uint, items []dto.BatchGetOntimeItem) dto.BatchGetOntimeResponse {

		result := lo.Map(items, func(item dto.BatchGetOntimeItem, _ int) dto.OntimeStats {
			return dto.OntimeStats{Date: item.Date, Stats: resultMap[item]}
		})

		return dto.BatchGetOntimeResponse{
			ServerID: serverID,
			Result:   result,
		}
	})
}
