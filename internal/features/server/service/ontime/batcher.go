package ontime

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/samber/lo/it"

	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository/ontime"
	featservice "github.com/minhnbnt/uptime-monitor/internal/features/server/service"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

func RegisterBatcher(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Batcher, error) {
		return &Batcher{
			serverRepository:      do.MustInvoke[*serverrepo.ServerRepository](i),
			ontimeCacheRepository: do.MustInvoke[*ontimerepo.OntimeCacheRepository](i),
			logger:                do.MustInvoke[logger.Logger](i),
			calculator:            OntimeCalculator{},
		}, nil
	})
}

type Batcher struct {
	serverRepository      featservice.ServerRepository
	ontimeCacheRepository OntimeCacheRepository
	logger                logger.Logger
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

	if err := b.ontimeCacheRepository.MSet(ctx, toCache); err != nil {
		b.logger.Warn("failed to batch cache ontime results", logger.Error(err))
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

	cached, err := b.ontimeCacheRepository.MGet(ctx, keys)

	if err != nil {
		b.logger.Warn("ontime cache MGet failed, falling back to DB", logger.Error(err))
		return make(map[dto.BatchGetOntimeItem]float64, len(keys))
	}

	return cached
}

func (b *Batcher) fillMisses(ctx context.Context, missedKeys []dto.BatchGetOntimeItem, until time.Time) map[dto.BatchGetOntimeItem]float64 {

	requests := lo.Map(missedKeys, func(key dto.BatchGetOntimeItem, _ int) serverrepo.BatchGetOntimeRequest {
		return serverrepo.BatchGetOntimeRequest{ServerID: key.ServerID, Date: key.Date}
	})

	rows, err := b.serverRepository.BatchGetOntime(ctx, requests)
	if err != nil {
		b.logger.Warn("failed to get missed ontime keys", logger.Error(err))
		return make(map[dto.BatchGetOntimeItem]float64)
	}

	groups := lo.GroupBy(rows, func(row serverrepo.RawEvent) serverDayKey {
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
