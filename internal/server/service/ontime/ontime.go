package ontime

import (
	"context"
	"fmt"
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
	serverRepository service.ServerRepository
	batcher          *Batcher
}

func RegisterOntimeService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeService, error) {
		return newOntimeService(
			do.MustInvoke[*serverrepo.ServerRepository](i),
			do.MustInvoke[*ontimerepo.OntimeCacheRepository](i),
			do.MustInvoke[logger.Logger](i),
		), nil
	})
}

func newOntimeService(serverRepo *serverrepo.ServerRepository, cacheRepo *ontimerepo.OntimeCacheRepository, log logger.Logger) *OntimeService {
	return &OntimeService{
		serverRepository: serverRepo,
		batcher: &Batcher{
			serverRepository:      serverRepo,
			ontimeCacheRepository: cacheRepo,
			logger:                log,
			calculator:            OntimeCalculator{},
		},
	}
}

type serverDayKey struct {
	ServerID uint
	Day      time.Time
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

	items := make([]dto.BatchGetOntimeItem, 0)
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

	results, err := s.batcher.BatchGetOntime(ctx, items)
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
