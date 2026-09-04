package service

import (
	"context"
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

type OntimeRepository interface {
	BatchGetUptime(ctx context.Context, req []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.UptimeRow, error)
}

type OntimeCacheRepository interface {
	MGet(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]dto.DayResult, error)
	MSet(ctx context.Context, items map[dto.BatchGetOntimeItem]dto.DayResult) error
}

func NewBatcher(repo OntimeRepository, cache *ontimerepo.OntimeCacheRepository, l *slog.Logger) *Batcher {

	var cacheInterface OntimeCacheRepository
	if cache != nil {
		cacheInterface = cache
	}

	return &Batcher{
		ontimeRepo:            repo,
		ontimeCacheRepository: cacheInterface,
		logger:                l,
	}
}

func RegisterBatcher(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Batcher, error) {
		return NewBatcher(
			do.MustInvoke[*ontimerepo.OntimeUptimeRepository](i),
			do.MustInvoke[*ontimerepo.OntimeCacheRepository](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

type Batcher struct {
	ontimeRepo            OntimeRepository
	ontimeCacheRepository OntimeCacheRepository
	logger                *slog.Logger
}

func (b *Batcher) BatchGetOntimeUntil(ctx context.Context, req []dto.BatchGetOntimeItem, until time.Time, loc *time.Location) ([]dto.BatchGetOntimeResponse, error) {

	if loc == nil {
		loc = time.UTC
	}
	cacheKeys := getCacheKey(req, loc)
	resultMap := b.resolveCache(ctx, cacheKeys)

	missKeys := lo.Filter(cacheKeys, func(key dto.BatchGetOntimeItem, _ int) bool {
		_, hit := resultMap[key]
		return !hit
	})

	if len(missKeys) == 0 {
		return b.buildResponse(req, resultMap), nil
	}

	toCache := b.fillMisses(ctx, missKeys, until, loc)
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
	return b.BatchGetOntimeUntil(ctx, req, time.Now(), time.UTC)
}

func getCacheKey(req []dto.BatchGetOntimeItem, loc *time.Location) []dto.BatchGetOntimeItem {

	if loc == nil {
		loc = time.UTC
	}
	reqIter := slices.Values(req)
	cacheKeys := it.Map(reqIter, func(item dto.BatchGetOntimeItem) dto.BatchGetOntimeItem {
		item.Date = utils.TruncateDayIn(item.Date, loc)
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

func (b *Batcher) fillMisses(ctx context.Context, missedKeys []dto.BatchGetOntimeItem, until time.Time, loc *time.Location) map[dto.BatchGetOntimeItem]dto.DayResult {

	if loc == nil {
		loc = time.UTC
	}
	requests := lo.Map(missedKeys, func(key dto.BatchGetOntimeItem, _ int) ontimerepo.BatchGetOntimeRequest {

		dayStart := key.Date // already truncated to midnight in loc by getCacheKey
		to := dayStart.Add(24 * time.Hour)

		// Same clamp semantics the old calculator had: only the calendar day
		// "until" falls on gets cut short at `until`; past days keep their
		// full 24h window.
		if utils.TruncateDayIn(until, loc).Equal(dayStart) && until.Before(to) {
			to = until
		}

		return ontimerepo.BatchGetOntimeRequest{EndpointID: key.EndpointID, From: dayStart, To: to}
	})

	rows, err := b.ontimeRepo.BatchGetUptime(ctx, requests)
	if err != nil {
		b.logger.Warn("failed to get missed ontime keys", slog.Any("error", err))
		// Return an empty map, not toCache: toCache is pre-seeded with zero
		// results for every missed key, so returning it would poison the
		// cache with bogus no-data/0% for a read that never completed.
		// Caller just re-fetches them next time.
		return make(map[dto.BatchGetOntimeItem]dto.DayResult)
	}

	// Pre-seed every missed key with a zero result so days with no events are
	// still cached (otherwise they'd be re-fetched on every request). The DB
	// only returns rows for keys that actually have data; everything else
	// keeps the zero value.
	toCache := lo.SliceToMap(missedKeys, func(key dto.BatchGetOntimeItem) (dto.BatchGetOntimeItem, dto.DayResult) {
		return key, dto.DayResult{}
	})

	for _, row := range rows {
		// TruncateDayIn normalizes both the zone and the wall clock: the DB
		// hands back timestamptz in the session zone, while missedKeys carry
		// midnight-in-loc dates — raw equality would never match.
		// Conversion is instant-based, so it holds for any session zone.
		key := dto.BatchGetOntimeItem{
			EndpointID: row.EndpointID,
			Date:       utils.TruncateDayIn(row.From, loc),
		}

		toCache[key] = dto.DayResult{
			HasData: row.HasData(),
			Uptime:  row.UptimePercent(),
			Unknown: row.UnknownSeconds,
		}
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
			return dto.OntimeStats{
				Date:           item.Date,
				Stats:          dr.Uptime,
				HasData:        dr.HasData,
				UnknownSeconds: dr.Unknown,
			}
		})

		return dto.BatchGetOntimeResponse{
			EndpointID: endpointID,
			Result:     result,
		}
	})
}
