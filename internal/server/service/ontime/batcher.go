package ontime

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/samber/lo/it"

	"github.com/minhnbnt/uptime-monitor/internal/logger"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/repository/ontime"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

func RegisterBatcher(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Batcher, error) {
		return &Batcher{
			serverRepository:      do.MustInvoke[service.ServerRepository](i),
			ontimeCacheRepository: do.MustInvoke[service.OntimeCacheRepository](i),
			logger:                do.MustInvoke[logger.Logger](i),
			calculator:            OntimeCalculator{},
		}, nil
	})
}

type Batcher struct {
	serverRepository      service.ServerRepository
	ontimeCacheRepository service.OntimeCacheRepository
	logger                logger.Logger
	calculator            OntimeCalculator
}

func (b *Batcher) BatchGetOntimeUntil(ctx context.Context, req []dto.BatchGetOntimeItem, until time.Time) ([]dto.BatchGetOntimeResponse, error) {

	cacheKeys := b.buildCacheKeys(req)
	resultMap := b.resolveCache(ctx, cacheKeys)

	missKeys := lo.Filter(cacheKeys, func(key ontimerepo.OntimeCacheKey, _ int) bool {
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

func (b *Batcher) buildCacheKeys(req []dto.BatchGetOntimeItem) []ontimerepo.OntimeCacheKey {

	iter := slices.Values(req)

	keys := it.Map(iter, func(item dto.BatchGetOntimeItem) ontimerepo.OntimeCacheKey {
		return ontimerepo.OntimeCacheKey{ServerID: item.ServerID, Day: utils.TruncateDay(item.Date)}
	})

	keys = it.Uniq(keys)
	return slices.Collect(keys)
}

func (b *Batcher) resolveCache(ctx context.Context, keys []ontimerepo.OntimeCacheKey) map[ontimerepo.OntimeCacheKey]float64 {

	cached, err := b.ontimeCacheRepository.MGet(ctx, keys)

	if err != nil {
		b.logger.Warn("ontime cache MGet failed, falling back to DB", logger.Error(err))
		return make(map[ontimerepo.OntimeCacheKey]float64, len(keys))
	}

	return cached
}

func (b *Batcher) fillMisses(ctx context.Context, missedKeys []ontimerepo.OntimeCacheKey, until time.Time) map[ontimerepo.OntimeCacheKey]float64 {

	requests := lo.Map(missedKeys, func(key ontimerepo.OntimeCacheKey, _ int) serverrepo.BatchGetOntimeRequest {
		return serverrepo.BatchGetOntimeRequest{ServerID: key.ServerID, Date: key.Day}
	})

	rows, err := b.serverRepository.BatchGetOntime(ctx, requests)
	if err != nil {
		b.logger.Warn("failed to get missed ontime keys", logger.Error(err))
		return make(map[ontimerepo.OntimeCacheKey]float64)
	}

	groups := lo.GroupBy(rows, func(row serverrepo.RawEvent) serverDayKey {
		return serverDayKey{ServerID: row.ServerID, Day: row.Day}
	})

	dayUntil := utils.TruncateDay(until)
	toCache := lo.SliceToMap(missedKeys, func(key ontimerepo.OntimeCacheKey) (ontimerepo.OntimeCacheKey, float64) {
		events := groups[serverDayKey{ServerID: key.ServerID, Day: key.Day}]
		return key, b.calculator.CalculateDayOntime(events, dayUntil, until)
	})

	return toCache
}

func (b *Batcher) buildResponse(req []dto.BatchGetOntimeItem, resultMap map[ontimerepo.OntimeCacheKey]float64) []dto.BatchGetOntimeResponse {

	serverResults := make(map[uint][]dto.OntimeStats)

	for _, item := range req {
		key := ontimerepo.OntimeCacheKey{ServerID: item.ServerID, Day: utils.TruncateDay(item.Date)}
		serverResults[item.ServerID] = append(serverResults[item.ServerID], dto.OntimeStats{
			Date:  key.Day,
			Stats: resultMap[key],
		})
	}

	responses := lo.MapToSlice(serverResults, func(serverID uint, result []dto.OntimeStats) dto.BatchGetOntimeResponse {
		return dto.BatchGetOntimeResponse{ServerID: serverID, Result: result}
	})

	return responses
}
