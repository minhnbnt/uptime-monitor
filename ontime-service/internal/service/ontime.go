package service

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/samber/lo/it"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
)

type OntimeService struct {
	serverOwnerRepo ServerOwnerRepository
	batcher         *Batcher
	logger          *slog.Logger
}

func NewOntimeService(ownerRepo ServerOwnerRepository, b *Batcher, l *slog.Logger) *OntimeService {
	return &OntimeService{
		serverOwnerRepo: ownerRepo,
		batcher:         b,
		logger:          l,
	}
}

func RegisterOntimeService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeService, error) {
		return NewOntimeService(
			do.MustInvoke[*repository.ServerOwnerRepository](i),
			do.MustInvoke[*Batcher](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

func (s *OntimeService) ListServersWithOntime(ctx context.Context, userID uint, page, perPage int) ([]dto.ServerOntime, error) {

	owned, err := s.serverOwnerRepo.ListByUser(ctx, userID)
	if err != nil {
		s.logger.Error("failed to list owned servers", slog.Any("error", err))
		return nil, err
	}

	total := len(owned)
	start := (page - 1) * perPage
	if start < 0 || start >= total {
		return []dto.ServerOntime{}, nil
	}
	end := start + perPage
	if end > total {
		end = total
	}
	pageOwned := owned[start:end]

	ontimeMap, err := s.getServersOntime(ctx, pageOwned, utils.Last30Days(), time.UTC)
	if err != nil {
		return nil, err
	}

	out := lo.Map(pageOwned, func(sv repository.OwnedServer, _ int) dto.ServerOntime {
		return dto.ServerOntime{
			ServerID:    sv.ServerID,
			OntimeStats: ontimeMap[sv.ServerID],
		}
	})

	return out, nil
}

func (s *OntimeService) GetServersOntime(ctx context.Context, userID uint, serverIDs []uint64, from, to time.Time) (map[uint][]dto.OntimeStats, error) {

	if len(serverIDs) == 0 {
		return make(map[uint][]dto.OntimeStats), nil
	}

	ids := lo.Map(serverIDs, func(id uint64, _ int) uint { return uint(id) })

	owned, err := s.serverOwnerRepo.GetOwnedServers(ctx, userID, ids)
	if err != nil {
		s.logger.Error(
			"failed to list owned servers for ontime",
			slog.Uint64("user_id", uint64(userID)),
			slog.Any("error", err),
		)
		return nil, err
	}
	if len(owned) == 0 {
		return make(map[uint][]dto.OntimeStats), nil
	}

	dates := utils.BuildDateRange(from, to)
	return s.getServersOntime(ctx, owned, dates, time.UTC)
}

func (s *OntimeService) GetServerWithOntime(ctx context.Context, serverID, userID uint) (*dto.ServerOntime, error) {

	owned, err := s.serverOwnerRepo.GetOwnedServers(ctx, userID, []uint{serverID})
	if err != nil {
		return nil, err
	}
	if len(owned) == 0 {
		return nil, apperrors.ErrNotFound
	}

	ontimeMap, err := s.getServersOntime(ctx, owned, utils.Last30Days(), time.UTC)
	if err != nil {
		return nil, err
	}

	return &dto.ServerOntime{
		ServerID:    serverID,
		OntimeStats: ontimeMap[serverID],
	}, nil
}

func (s *OntimeService) GetServersWithOntime(ctx context.Context, userID uint, ids []uint, loc *time.Location) ([]dto.ServerOntime, error) {

	if loc == nil {
		loc = time.UTC
	}

	owned, err := s.serverOwnerRepo.GetOwnedServers(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	if len(owned) != len(ids) {
		return nil, apperrors.ErrForbidden
	}

	ontimeMap, err := s.getServersOntime(ctx, owned, utils.Last30DaysIn(loc), loc)
	if err != nil {
		return nil, err
	}

	out := lo.Map(owned, func(sv repository.OwnedServer, _ int) dto.ServerOntime {
		return dto.ServerOntime{
			OntimeStats: ontimeMap[sv.ServerID],
			ServerID:    sv.ServerID,
		}
	})

	return out, nil
}

func (s *OntimeService) getServersOntime(ctx context.Context, servers []repository.OwnedServer, dates []time.Time, loc *time.Location) (map[uint][]dto.OntimeStats, error) {

	if loc == nil {
		loc = time.UTC
	}
	if len(dates) == 0 {
		dates = utils.Last30DaysIn(loc)
	}

	items := make([]dto.BatchGetOntimeItem, 0, len(servers)*len(dates))
	serverDates := make(map[uint][]time.Time, len(servers))

	for _, sv := range servers {

		created := utils.TruncateDayIn(sv.CreatedAt, loc)
		activeDates := lo.Filter(dates, func(d time.Time, _ int) bool {
			return !d.Before(created)
		})

		datesIter := slices.Values(activeDates)
		newItems := it.Map(datesIter, func(d time.Time) dto.BatchGetOntimeItem {
			return dto.BatchGetOntimeItem{EndpointID: sv.ServerID, Date: d}
		})
		items = slices.AppendSeq(items, newItems)
		serverDates[sv.ServerID] = activeDates
	}

	if len(items) == 0 {
		return make(map[uint][]dto.OntimeStats), nil
	}

	results, err := s.batcher.BatchGetOntimeUntil(ctx, items, time.Now(), loc)
	if err != nil {
		s.logger.Error("failed to batch get ontime", slog.Any("error", err))
		return nil, err
	}

	lookup := buildOntimeLookup(results, loc)

	out := make(map[uint][]dto.OntimeStats, len(servers))
	for _, sv := range servers {

		stats, ok := lookup[sv.ServerID]
		if !ok {
			stats = make(map[time.Time]dto.DayResult)
		}

		out[sv.ServerID] = lo.Map(serverDates[sv.ServerID], func(d time.Time, _ int) dto.OntimeStats {
			dr := stats[d]
			return dto.OntimeStats{
				Date:           d,
				Stats:          dr.Uptime,
				HasData:        dr.HasData,
				UnknownSeconds: dr.Unknown,
			}
		})
	}

	return out, nil
}

func buildOntimeLookup(results []dto.BatchGetOntimeResponse, loc *time.Location) map[uint]map[time.Time]dto.DayResult {

	if loc == nil {
		loc = time.UTC
	}
	lookup := make(map[uint]map[time.Time]dto.DayResult, len(results))

	for _, r := range results {

		mp := lo.SliceToMap(r.Result, func(stat dto.OntimeStats) (time.Time, dto.DayResult) {

			dayResult := dto.DayResult{
				HasData: stat.HasData,
				Uptime:  stat.Stats,
				Unknown: stat.UnknownSeconds,
			}

			return utils.TruncateDayIn(stat.Date, loc), dayResult
		})

		lookup[r.EndpointID] = mp
	}

	return lookup
}
