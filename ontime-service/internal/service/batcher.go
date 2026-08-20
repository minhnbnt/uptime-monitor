package service

import (
	"context"
	"iter"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/samber/lo/it"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
)

type OntineRepository interface {
	BatchGetOntime(ctx context.Context, req []ontimerepo.BatchGetOntimeRequest) (iter.Seq2[ontimerepo.RawEvent, error], error)
}

type OntimeCacheRepository interface {
	MGet(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]dto.DayResult, error)
	MSet(ctx context.Context, items map[dto.BatchGetOntimeItem]dto.DayResult) error
}

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

	if b.ontimeCacheRepository == nil {
		return b.buildResponse(req, resultMap), nil
	}

	if err := b.ontimeCacheRepository.MSet(ctx, toCache); err != nil {
		b.logger.Warn("failed to batch cache ontime results", slog.Any("error", err))
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

func (b *Batcher) resolveCache(ctx context.Context, keys []dto.BatchGetOntimeItem) map[dto.BatchGetOntimeItem]dto.DayResult {

	if b.ontimeCacheRepository == nil {
		return make(map[dto.BatchGetOntimeItem]dto.DayResult, len(keys))
	}

	cached, err := b.ontimeCacheRepository.MGet(ctx, keys)
	if err != nil {
		b.logger.Warn("ontime cache MGet failed, falling back to DB", slog.Any("error", err))
		return make(map[dto.BatchGetOntimeItem]dto.DayResult, len(keys))
	}

	return cached
}

func (b *Batcher) fillMisses(ctx context.Context, missedKeys []dto.BatchGetOntimeItem, until time.Time) map[dto.BatchGetOntimeItem]dto.DayResult {

	requests := lo.Map(missedKeys, func(key dto.BatchGetOntimeItem, _ int) ontimerepo.BatchGetOntimeRequest {
		return ontimerepo.BatchGetOntimeRequest{EndpointID: key.EndpointID, Date: key.Date}
	})

	seq, err := b.ontineRepository.BatchGetOntime(ctx, requests)
	if err != nil {
		b.logger.Warn("failed to get missed ontime keys", slog.Any("error", err))
		return make(map[dto.BatchGetOntimeItem]dto.DayResult)
	}

	// Pre-seed every missed key with a zero result so days with no events are
	// still cached (otherwise they'd be re-fetched on every request).
	toCache := lo.SliceToMap(missedKeys, func(key dto.BatchGetOntimeItem) (dto.BatchGetOntimeItem, dto.DayResult) {
		return key, dto.DayResult{}
	})

	// ponytail: the SQL orders by (endpoint_id, day, time), so we finalize a
	// group the instant the key changes and discard its events — peak RAM is one
	// day, not the whole result set. Each group is computed over its OWN day
	// window (utils.TruncateDay(k.Day)), not "today" — otherwise past days
	// resolve against the wrong 24h window and report 0%.
	var curEvents []ontimerepo.RawEvent
	flush := func(k endpointDayKey, events []ontimerepo.RawEvent) {

		today := utils.TruncateDay(k.Day)
		result := b.calculator.CalculateDayOntime(events, today, until)

		itemKey := dto.BatchGetOntimeItem{EndpointID: k.EndpointID, Date: k.Day}
		toCache[itemKey] = dto.DayResult{HasData: result.HasData, Uptime: result.Uptime}
	}

	curKey := endpointDayKey{}
	for row, err := range seq {

		if err != nil {
			b.logger.Warn("failed to stream missed ontime rows", slog.Any("error", err))
			// Return an empty map, not toCache: toCache is pre-seeded with zero
			// results for every missed key, so returning it would poison the
			// cache with bogus no-data/0% for days we never finished reading.
			// Caller just re-fetches them next time.
			return make(map[dto.BatchGetOntimeItem]dto.DayResult)
		}

		k := endpointDayKey{EndpointID: row.EndpointID, Day: row.Day}
		if k != curKey && len(curEvents) > 0 {
			flush(curKey, curEvents)
			curEvents = nil
		}

		curKey = k
		curEvents = append(curEvents, row)
	}

	if len(curEvents) > 0 {
		flush(curKey, curEvents)
	}

	return toCache
}

func (b *Batcher) buildResponse(req []dto.BatchGetOntimeItem, resultMap map[dto.BatchGetOntimeItem]dto.DayResult) []dto.BatchGetOntimeResponse {

	groups := lo.GroupBy(req, func(item dto.BatchGetOntimeItem) uint {
		return item.EndpointID
	})

	return lo.MapToSlice(groups, func(endpointID uint, items []dto.BatchGetOntimeItem) dto.BatchGetOntimeResponse {

		result := lo.Map(items, func(item dto.BatchGetOntimeItem, _ int) dto.OntimeStats {
			dr := resultMap[item]
			return dto.OntimeStats{Date: item.Date, Stats: dr.Uptime, HasData: dr.HasData}
		})

		return dto.BatchGetOntimeResponse{
			EndpointID: endpointID,
			Result:     result,
		}
	})
}

type endpointDayKey struct {
	EndpointID uint
	Day        time.Time
}
