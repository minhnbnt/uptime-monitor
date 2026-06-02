package service

import (
	"context"
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
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func (s *OntimeService) BatchGetOntime(ctx context.Context, req []dto.BatchGetOntimeItem) ([]dto.BatchGetOntimeResponse, error) {

	cacheKeys := s.buildCacheKeys(req)
	resultMap := s.resolveCache(ctx, cacheKeys)
	s.fillMisses(ctx, resultMap, cacheKeys)

	return s.buildResponse(req, resultMap), nil
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
