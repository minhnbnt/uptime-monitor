package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	repo "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

type OntimeService struct {
	serverRepository      ServerRepository
	ontimeCacheRepository OntimeCacheRepository
	logger                logger.Logger
	calculator            OntimeCalculator
}

func RegisterOntimeService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeService, error) {
		return &OntimeService{
			serverRepository:      do.MustInvoke[*repo.ServerRepository](i),
			ontimeCacheRepository: do.MustInvoke[*repo.OntimeCacheRepository](i),
			logger:                do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

type serverDayKey struct {
	ServerID uint
	Day      time.Time
}

func (s *OntimeService) BatchGetOntime(ctx context.Context, req []dto.BatchGetOntimeItem) ([]dto.BatchGetOntimeResponse, error) {

	cacheKeys := s.buildCacheKeys(req)
	resultMap := s.resolveCache(ctx, cacheKeys)

	s.fillMisses(ctx, resultMap, cacheKeys)

	return s.buildResponse(req, resultMap), nil
}

func (s *OntimeService) ListServersWithOntime(ctx context.Context, page, perPage int) ([]dto.ServerWithOntime, int64, error) {

	dates := utils.Last30Days()
	servers, err := s.serverRepository.List(ctx, perPage, (page-1)*perPage)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list servers: %w", err)
	}

	total, err := s.serverRepository.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count servers: %w", err)
	}

	items := make([]dto.BatchGetOntimeItem, 0, len(servers)*len(dates))
	serverDates := make(map[uint][]time.Time, len(servers))

	for _, sv := range servers {
		created := utils.TruncateDay(sv.CreatedAt)
		for _, d := range dates {
			if d.Before(created) {
				continue
			}
			items = append(items, dto.BatchGetOntimeItem{
				ServerID: sv.ID,
				Date:     d,
			})
			serverDates[sv.ID] = append(serverDates[sv.ID], d)
		}
	}

	results, err := s.BatchGetOntime(ctx, items)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to batch get ontime: %w", err)
	}

	lookup := buildOntimeLookup(results)
	out := make([]dto.ServerWithOntime, 0, len(servers))
	for _, sv := range servers {

		stats, ok := lookup[sv.ID]
		if !ok {
			stats = make(map[time.Time]float64)
		}

		otStats := lo.Map(serverDates[sv.ID], func(d time.Time, _ int) dto.OntimeStats {
			return dto.OntimeStats{Date: d, Stats: stats[d]}
		})

		out = append(out, dto.ServerWithOntime{
			Server:      toDTOServer(sv),
			OntimeStats: otStats,
		})
	}

	return out, total, nil
}

func buildOntimeLookup(results []dto.BatchGetOntimeResponse) map[uint]map[time.Time]float64 {

	lookup := make(map[uint]map[time.Time]float64, len(results))

	for _, r := range results {

		mp := lo.SliceToMap(r.Result, func(stat dto.OntimeStats) (time.Time, float64) {
			return utils.TruncateDay(stat.Date), stat.Stats
		})

		lookup[r.ServerID] = mp
	}

	return lookup
}

func (s *OntimeService) buildCacheKeys(req []dto.BatchGetOntimeItem) []repo.OntimeCacheKey {

	keys := lo.Map(req, func(item dto.BatchGetOntimeItem, _ int) repo.OntimeCacheKey {
		return repo.OntimeCacheKey{ServerID: item.ServerID, Day: utils.TruncateDay(item.Date)}
	})

	return lo.Uniq(keys)
}

func (s *OntimeService) resolveCache(ctx context.Context, keys []repo.OntimeCacheKey) map[repo.OntimeCacheKey]float64 {

	cached, err := s.ontimeCacheRepository.MGet(ctx, keys)

	if err != nil {
		s.logger.Warn("ontime cache MGet failed, falling back to DB", logger.Error(err))
		return make(map[repo.OntimeCacheKey]float64, len(keys))
	}

	return cached
}

func (s *OntimeService) fillMisses(ctx context.Context, resultMap map[repo.OntimeCacheKey]float64, cacheKeys []repo.OntimeCacheKey) {

	missKeys := lo.Filter(cacheKeys, func(key repo.OntimeCacheKey, _ int) bool {
		_, has := resultMap[key]
		return !has
	})

	if len(missKeys) == 0 {
		return
	}

	requests := lo.Map(missKeys, func(key repo.OntimeCacheKey, _ int) repo.BatchGetOntimeRequest {
		return repo.BatchGetOntimeRequest{ServerID: key.ServerID, Date: key.Day}
	})

	rows, err := s.serverRepository.BatchGetOntime(ctx, requests)
	if err != nil {
		s.logger.Warn("failed to batch get ontime from DB", logger.Error(err))
		return
	}

	groups := lo.GroupBy(rows, func(row repo.RawEvent) serverDayKey {
		return serverDayKey{ServerID: row.ServerID, Day: row.Day}
	})

	now := time.Now()
	today := utils.TruncateDay(now)
	toCache := make(map[repo.OntimeCacheKey]float64, len(missKeys))

	for _, key := range missKeys {

		events := groups[serverDayKey{ServerID: key.ServerID, Day: key.Day}]
		stats := s.calculator.CalculateDayOntime(events, today, now)

		resultMap[key] = stats
		toCache[key] = stats
	}

	if err := s.ontimeCacheRepository.MSet(ctx, toCache); err != nil {
		s.logger.Warn("failed to batch cache ontime results", logger.Error(err))
	}
}

func (s *OntimeService) buildResponse(req []dto.BatchGetOntimeItem, resultMap map[repo.OntimeCacheKey]float64) []dto.BatchGetOntimeResponse {

	serverResults := make(map[uint][]dto.OntimeStats)

	for _, item := range req {
		key := repo.OntimeCacheKey{ServerID: item.ServerID, Day: utils.TruncateDay(item.Date)}
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
