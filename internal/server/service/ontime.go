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
)

type OntimeService struct {
	repo   *repo.ServerRepository
	cache  *repo.OntimeCacheRepository
	logger logger.Logger
}

func RegisterOntimeService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeService, error) {
		return &OntimeService{
			repo:   do.MustInvoke[*repo.ServerRepository](i),
			cache:  do.MustInvoke[*repo.OntimeCacheRepository](i),
			logger: do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *OntimeService) BatchGetOntime(ctx context.Context, req []dto.BatchGetOntimeItem) ([]dto.BatchGetOntimeResponse, error) {

	cacheKeys := s.buildCacheKeys(req)
	resultMap := s.resolveCache(ctx, cacheKeys)
	s.fillMisses(ctx, resultMap, cacheKeys)

	return s.buildResponse(req, resultMap), nil
}

func (s *OntimeService) ListServersWithOntime(ctx context.Context, page, perPage int) ([]dto.ServerWithOntime, int64, error) {
	since := truncateDay(time.Now().AddDate(0, 0, -29))
	until := truncateDay(time.Now())

	var dates []time.Time
	for d := since; !d.After(until); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d)
	}

	servers, err := s.repo.List(ctx, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list servers: %w", err)
	}

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count servers: %w", err)
	}

	items := make([]dto.BatchGetOntimeItem, 0, len(servers)*len(dates))
	for _, sv := range servers {
		for _, d := range dates {
			items = append(items, dto.BatchGetOntimeItem{ServerID: sv.ID, Date: d})
		}
	}

	ontimeResults, err := s.BatchGetOntime(ctx, items)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to batch get ontime: %w", err)
	}

	lookup := make(map[uint]map[time.Time]float64, len(ontimeResults))
	for _, r := range ontimeResults {
		statsMap := make(map[time.Time]float64, len(r.Result))
		for _, stat := range r.Result {
			statsMap[truncateDay(stat.Date)] = stat.Stats
		}
		lookup[r.ServerID] = statsMap
	}

	result := make([]dto.ServerWithOntime, 0, len(servers))
	for _, sv := range servers {
		otStats := make([]dto.OntimeStats, 0, len(dates))
		for _, d := range dates {
			stats := lookup[sv.ID][d]
			otStats = append(otStats, dto.OntimeStats{Date: d, Stats: stats})
		}
		result = append(result, dto.ServerWithOntime{
			Server:      toDTOServer(sv),
			OntimeStats: otStats,
		})
	}

	return result, total, nil
}

func (s *OntimeService) buildCacheKeys(req []dto.BatchGetOntimeItem) []repo.OntimeCacheKey {

	cacheKeys := lo.Map(req, func(item dto.BatchGetOntimeItem, _ int) repo.OntimeCacheKey {
		return repo.OntimeCacheKey{ServerID: item.ServerID, Day: truncateDay(item.Date)}
	})

	return lo.Uniq(cacheKeys)
}

func (s *OntimeService) resolveCache(ctx context.Context, keys []repo.OntimeCacheKey) map[repo.OntimeCacheKey]float64 {

	resultMap := make(map[repo.OntimeCacheKey]float64, len(keys))

	cached, err := s.cache.MGet(ctx, keys)
	if err != nil {
		s.logger.Warn("ontime cache MGet failed, falling back to DB", logger.Error(err))
		return resultMap
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
	dbResults, err := s.repo.BatchGetOntime(ctx, requests)
	if err != nil {
		s.logger.Warn("failed to batch get ontime from DB", logger.Error(err))
		return
	}

	toCache := make(map[repo.OntimeCacheKey]float64, len(dbResults))
	for _, dbRes := range dbResults {
		for _, r := range dbRes.Result {

			key := repo.OntimeCacheKey{
				ServerID: dbRes.ServerID,
				Day:      truncateDay(r.Date),
			}

			toCache[key] = r.Stats
			resultMap[key] = r.Stats
		}
	}

	if err := s.cache.MSet(ctx, toCache); err != nil {
		s.logger.Warn("failed to batch cache ontime results", logger.Error(err))
	}
}

func (s *OntimeService) buildResponse(req []dto.BatchGetOntimeItem, resultMap map[repo.OntimeCacheKey]float64) []dto.BatchGetOntimeResponse {

	serverResults := make(map[uint][]dto.OntimeStats)
	for _, item := range req {

		key := repo.OntimeCacheKey{
			ServerID: item.ServerID,
			Day:      truncateDay(item.Date),
		}

		serverResults[item.ServerID] = append(
			serverResults[item.ServerID],
			dto.OntimeStats{
				Date:  key.Day,
				Stats: resultMap[key],
			},
		)
	}

	responses := make([]dto.BatchGetOntimeResponse, 0, len(serverResults))
	for serverID, result := range serverResults {
		responses = append(responses, dto.BatchGetOntimeResponse{ServerID: serverID, Result: result})
	}

	return responses
}
