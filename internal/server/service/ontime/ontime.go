package ontime

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/repository/ontime"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

type OntimeService struct {
	serverRepository      service.ServerRepository
	ontimeCacheRepository service.OntimeCacheRepository
	logger                logger.Logger
	calculator            OntimeCalculator
}

func RegisterOntimeService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeService, error) {
		return &OntimeService{
			serverRepository:      do.MustInvoke[*serverrepo.ServerRepository](i),
			ontimeCacheRepository: do.MustInvoke[*ontimerepo.OntimeCacheRepository](i),
			logger:                do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

type serverDayKey struct {
	ServerID uint
	Day      time.Time
}

func (s *OntimeService) BatchGetOntimeUntil(ctx context.Context, req []dto.BatchGetOntimeItem, until time.Time) ([]dto.BatchGetOntimeResponse, error) {

	cacheKeys := s.buildCacheKeys(req)
	resultMap := s.resolveCache(ctx, cacheKeys)

	missKeys := lo.Filter(cacheKeys, func(key ontimerepo.OntimeCacheKey, _ int) bool {
		_, hit := resultMap[key]
		return !hit
	})

	if len(missKeys) == 0 {
		return s.buildResponse(req, resultMap), nil
	}

	toCache := s.fillMisses(ctx, missKeys, until)
	maps.Copy(resultMap, toCache)

	if err := s.ontimeCacheRepository.MSet(ctx, toCache); err != nil {
		s.logger.Warn("failed to batch cache ontime results", logger.Error(err))
	}

	return s.buildResponse(req, resultMap), nil
}

func (s *OntimeService) BatchGetOntime(ctx context.Context, req []dto.BatchGetOntimeItem) ([]dto.BatchGetOntimeResponse, error) {
	return s.BatchGetOntimeUntil(ctx, req, time.Now())
}

func (s *OntimeService) ListServersWithOntime(ctx context.Context, createdByID uint, page, perPage int) ([]dto.ServerWithOntime, int64, error) {

	servers, err := s.serverRepository.List(ctx, createdByID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list servers: %w", err)
	}

	total, err := s.serverRepository.Count(ctx, createdByID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count servers: %w", err)
	}

	ontimeMap, err := s.getServersOntime(ctx, servers)
	if err != nil {
		return nil, 0, err
	}

	out := lo.Map(servers, func(sv domain.Server, _ int) dto.ServerWithOntime {
		return dto.ServerWithOntime{
			Server:      dto.ServerFromDomain(sv),
			OntimeStats: ontimeMap[sv.ID],
		}
	})

	return out, total, nil
}

func (s *OntimeService) GetServerWithOntime(ctx context.Context, serverID uint) (*dto.ServerWithOntime, error) {

	server, err := s.serverRepository.GetByID(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	ontimeMap, err := s.getServersOntime(ctx, []domain.Server{*server})
	if err != nil {
		return nil, err
	}

	dtoSrv := dto.ServerFromDomain(*server)
	return &dto.ServerWithOntime{
		Server:      dtoSrv,
		OntimeStats: ontimeMap[server.ID],
	}, nil
}

func (s *OntimeService) getServersOntime(ctx context.Context, servers []domain.Server) (map[uint][]dto.OntimeStats, error) {

	dates := utils.Last30Days()

	items := make([]dto.BatchGetOntimeItem, 0, len(servers)*len(dates))
	serverDates := make(map[uint][]time.Time, len(servers))

	for _, sv := range servers {

		created := utils.TruncateDay(sv.CreatedAt)
		dates := lo.Filter(dates, func(d time.Time, _ int) bool {
			return !d.Before(created)
		})

		newItems := lo.Map(dates, func(d time.Time, _ int) dto.BatchGetOntimeItem {
			return dto.BatchGetOntimeItem{ServerID: sv.ID, Date: d}
		})

		items = append(items, newItems...)
		serverDates[sv.ID] = dates
	}

	if len(items) == 0 {
		return make(map[uint][]dto.OntimeStats), nil
	}

	results, err := s.BatchGetOntime(ctx, items)
	if err != nil {
		return nil, fmt.Errorf("failed to batch get ontime: %w", err)
	}

	lookup := buildOntimeLookup(results)

	out := make(map[uint][]dto.OntimeStats, len(servers))
	for _, sv := range servers {

		stats, ok := lookup[sv.ID]
		if !ok {
			stats = make(map[time.Time]float64)
		}

		out[sv.ID] = lo.Map(serverDates[sv.ID], func(d time.Time, _ int) dto.OntimeStats {
			return dto.OntimeStats{Date: d, Stats: stats[d]}
		})
	}

	return out, nil
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

func (s *OntimeService) buildCacheKeys(req []dto.BatchGetOntimeItem) []ontimerepo.OntimeCacheKey {

	keys := lo.Map(req, func(item dto.BatchGetOntimeItem, _ int) ontimerepo.OntimeCacheKey {
		return ontimerepo.OntimeCacheKey{ServerID: item.ServerID, Day: utils.TruncateDay(item.Date)}
	})

	return lo.Uniq(keys)
}

func (s *OntimeService) resolveCache(ctx context.Context, keys []ontimerepo.OntimeCacheKey) map[ontimerepo.OntimeCacheKey]float64 {

	cached, err := s.ontimeCacheRepository.MGet(ctx, keys)

	if err != nil {
		s.logger.Warn("ontime cache MGet failed, falling back to DB", logger.Error(err))
		return make(map[ontimerepo.OntimeCacheKey]float64, len(keys))
	}

	return cached
}

func (s *OntimeService) fillMisses(ctx context.Context, missedKeys []ontimerepo.OntimeCacheKey, until time.Time) map[ontimerepo.OntimeCacheKey]float64 {

	requests := lo.Map(missedKeys, func(key ontimerepo.OntimeCacheKey, _ int) serverrepo.BatchGetOntimeRequest {
		return serverrepo.BatchGetOntimeRequest{ServerID: key.ServerID, Date: key.Day}
	})

	rows, err := s.serverRepository.BatchGetOntime(ctx, requests)
	if err != nil {
		s.logger.Warn("failed to get missed ontime keys", logger.Error(err))
		return make(map[ontimerepo.OntimeCacheKey]float64)
	}

	groups := lo.GroupBy(rows, func(row serverrepo.RawEvent) serverDayKey {
		return serverDayKey{ServerID: row.ServerID, Day: row.Day}
	})

	dayUntil := utils.TruncateDay(until)
	toCache := lo.SliceToMap(missedKeys, func(key ontimerepo.OntimeCacheKey) (ontimerepo.OntimeCacheKey, float64) {
		events := groups[serverDayKey{ServerID: key.ServerID, Day: key.Day}]
		return key, s.calculator.CalculateDayOntime(events, dayUntil, until)
	})

	return toCache
}

func (s *OntimeService) buildResponse(req []dto.BatchGetOntimeItem, resultMap map[ontimerepo.OntimeCacheKey]float64) []dto.BatchGetOntimeResponse {

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
