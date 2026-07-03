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
	ontimedto "github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
	serverdto "github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/service"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

type OntimeService struct {
	serverRepository service.ServerRepository
	batcher          *Batcher
	logger           logger.Logger
}

func NewOntimeService(sr service.ServerRepository, b *Batcher, l logger.Logger) *OntimeService {
	return &OntimeService{
		serverRepository: sr,
		batcher:          b,
		logger:           l,
	}
}

func RegisterOntimeService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeService, error) {
		return NewOntimeService(
			do.MustInvoke[*serverrepo.ServerRepository](i),
			do.MustInvoke[*Batcher](i),
			do.MustInvoke[logger.Logger](i),
		), nil
	})
}

type serverDayKey struct {
	ServerID uint
	Day      time.Time
}

func (s *OntimeService) ListServersWithOntime(ctx context.Context, createdByID uint, page, perPage int) ([]ontimedto.ServerWithOntime, int64, error) {

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

	out := lo.Map(servers, func(sv domain.Server, _ int) ontimedto.ServerWithOntime {
		return ontimedto.ServerWithOntime{
			Server:      serverdto.ServerFromDomain(sv),
			OntimeStats: ontimeMap[sv.ID],
		}
	})

	return out, total, nil
}

func (s *OntimeService) GetServerWithOntime(ctx context.Context, serverID uint, userID uint) (*ontimedto.ServerWithOntime, error) {

	server, err := s.serverRepository.GetByID(ctx, serverID)
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.ErrNotFound
	}
	if err != nil {
		s.logger.Error("failed to get server", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	if server.CreatedByID != userID {
		return nil, apperrors.ErrForbidden
	}

	ontimeMap, err := s.getServersOntime(ctx, []domain.Server{*server})
	if err != nil {
		return nil, err
	}

	dtoSrv := serverdto.ServerFromDomain(*server)
	return &ontimedto.ServerWithOntime{
		Server:      dtoSrv,
		OntimeStats: ontimeMap[server.ID],
	}, nil
}

func (s *OntimeService) getServersOntime(ctx context.Context, servers []domain.Server) (map[uint][]ontimedto.OntimeStats, error) {
	return s.GetServersOntimeForDates(ctx, servers, utils.Last30Days())
}

func (s *OntimeService) GetServersOntimeForDates(ctx context.Context, servers []domain.Server, dates []time.Time) (map[uint][]ontimedto.OntimeStats, error) {

	items := make([]ontimedto.BatchGetOntimeItem, 0, len(servers)*len(dates))
	serverDates := make(map[uint][]time.Time, len(servers))

	for _, sv := range servers {

		created := utils.TruncateDay(sv.CreatedAt)
		dates := lo.Filter(dates, func(d time.Time, _ int) bool {
			return !d.Before(created)
		})

		datesIter := slices.Values(dates)

		newItems := it.Map(datesIter, func(d time.Time) ontimedto.BatchGetOntimeItem {
			return ontimedto.BatchGetOntimeItem{ServerID: sv.ID, Date: d}
		})

		items = slices.AppendSeq(items, newItems)
		serverDates[sv.ID] = dates
	}

	if len(items) == 0 {
		return make(map[uint][]ontimedto.OntimeStats), nil
	}

	results, err := s.batcher.BatchGetOntime(ctx, items)
	if err != nil {
		s.logger.Error("failed to batch get ontime", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	lookup := buildOntimeLookup(results)

	out := make(map[uint][]ontimedto.OntimeStats, len(servers))
	for _, sv := range servers {

		stats, ok := lookup[sv.ID]
		if !ok {
			stats = make(map[time.Time]float64)
		}

		out[sv.ID] = lo.Map(serverDates[sv.ID], func(d time.Time, _ int) ontimedto.OntimeStats {
			return ontimedto.OntimeStats{Date: d, Stats: stats[d]}
		})
	}

	return out, nil
}

func buildOntimeLookup(results []ontimedto.BatchGetOntimeResponse) map[uint]map[time.Time]float64 {

	lookup := make(map[uint]map[time.Time]float64, len(results))

	for _, r := range results {

		mp := lo.SliceToMap(r.Result, func(stat ontimedto.OntimeStats) (time.Time, float64) {
			return utils.TruncateDay(stat.Date), stat.Stats
		})

		lookup[r.ServerID] = mp
	}

	return lookup
}
