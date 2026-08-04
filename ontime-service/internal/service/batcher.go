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

type OntineRepository interface {
	BatchGetOntime(ctx context.Context, req []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.ServerEvent, error)
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

func (b *Batcher) BatchGetOntimeUntil(ctx context.Context, req []dto.BatchGetOntimeItem, until time.Time) ([]dto.ServerOntime, error) {

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

func (b *Batcher) BatchGetOntime(ctx context.Context, req []dto.BatchGetOntimeItem) ([]dto.ServerOntime, error) {
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
		return ontimerepo.BatchGetOntimeRequest{ServerID: key.ServerID, Date: key.Date}
	})

	rows, err := b.ontineRepository.BatchGetOntime(ctx, requests)
	if err != nil {
		b.logger.Warn("failed to get missed ontime keys", slog.Any("error", err))
		return make(map[dto.BatchGetOntimeItem]dto.DayResult)
	}

	groups := lo.GroupBy(rows, func(row ontimerepo.ServerEvent) serverDayKey {
		return serverDayKey{ServerID: row.ServerID, Day: row.Event.AnchorTime}
	})

	toCache := lo.SliceToMap(missedKeys, func(key dto.BatchGetOntimeItem) (dto.BatchGetOntimeItem, dto.DayResult) {

		serverEvents := groups[serverDayKey{ServerID: key.ServerID, Day: key.Date}]
		events := lo.Map(serverEvents, func(se ontimerepo.ServerEvent, _ int) ontimerepo.Event {
			return se.Event
		})

		result := b.calculator.CalculateDayOntime(events, utils.TruncateDay(key.Date), until)
		return key, dto.DayResult{HasData: result.HasData, Uptime: result.Uptime}
	})

	return toCache
}

func (b *Batcher) buildResponse(req []dto.BatchGetOntimeItem, resultMap map[dto.BatchGetOntimeItem]dto.DayResult) []dto.ServerOntime {

	groups := lo.GroupBy(req, func(item dto.BatchGetOntimeItem) uint {
		return item.ServerID
	})

	return lo.MapToSlice(groups, func(serverID uint, items []dto.BatchGetOntimeItem) dto.ServerOntime {

		result := lo.Map(items, func(item dto.BatchGetOntimeItem, _ int) dto.DayStats {
			r := resultMap[item] // zero value -> HasData: false, correctly means "no data" if truly absent too
			return dto.DayStats{Date: item.Date, Result: r}
		})

		return dto.ServerOntime{
			ServerID: serverID,
			DayStats: result,
		}
	})
}

type serverDayKey struct {
	ServerID uint
	Day      time.Time
}
