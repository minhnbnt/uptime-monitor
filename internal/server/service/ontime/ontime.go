package ontime

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/samber/lo/it"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

type OntimeService struct {
	serverRepository service.ServerRepository
	batcher          *Batcher
	logger           logger.Logger
}

func RegisterOntimeService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeService, error) {
		return &OntimeService{
			serverRepository: do.MustInvoke[*serverrepo.ServerRepository](i),
			batcher:          do.MustInvoke[*Batcher](i),
			logger:           do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

type serverDayKey struct {
	ServerID uint
	Day      time.Time
}

func (s *OntimeService) ListServersWithOntime(ctx context.Context, createdByID uint, page, perPage int) ([]dto.ServerWithOntime, int64, error) {

	servers, err := s.serverRepository.List(ctx, createdByID, perPage, (page-1)*perPage)
	if err != nil {
		s.logger.Error("failed to list servers", logger.Error(err))
		return nil, 0, apperrors.ErrInternal
	}

	total, err := s.serverRepository.Count(ctx, createdByID)
	if err != nil {
		s.logger.Error("failed to count servers", logger.Error(err))
		return nil, 0, apperrors.ErrInternal
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
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.ErrNotFound
	}
	if err != nil {
		s.logger.Error("failed to get server", logger.Error(err))
		return nil, apperrors.ErrInternal
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

		datesIter := slices.Values(dates)

		newItems := it.Map(datesIter, func(d time.Time) dto.BatchGetOntimeItem {
			return dto.BatchGetOntimeItem{ServerID: sv.ID, Date: d}
		})

		items = slices.AppendSeq(items, newItems)
		serverDates[sv.ID] = dates
	}

	if len(items) == 0 {
		return make(map[uint][]dto.OntimeStats), nil
	}

	results, err := s.batcher.BatchGetOntime(ctx, items)
	if err != nil {
		s.logger.Error("failed to batch get ontime", logger.Error(err))
		return nil, apperrors.ErrInternal
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
