package service

import (
	"context"
	"log/slog"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/samber/lo/it"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/serverclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
)

type ServerClient interface {
	ListServers(ctx context.Context, userID uuid.UUID, page, perPage int) ([]serverclient.ServerBrief, error)
	GetServer(ctx context.Context, serverID uint, userID uuid.UUID) (*serverclient.ServerBrief, error)
}

type OntimeService struct {
	serverClient ServerClient
	batcher      *Batcher
	logger       *slog.Logger
}

func NewOntimeService(sc ServerClient, b *Batcher, l *slog.Logger) *OntimeService {
	return &OntimeService{
		serverClient: sc,
		batcher:      b,
		logger:       l,
	}
}

func RegisterOntimeService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeService, error) {
		return NewOntimeService(
			do.MustInvoke[*serverclient.Client](i),
			do.MustInvoke[*Batcher](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

func (s *OntimeService) ListServersWithOntime(ctx context.Context, userID uuid.UUID, page, perPage int) ([]dto.ServerOntime, error) {

	servers, err := s.serverClient.ListServers(ctx, userID, page, perPage)
	if err != nil {
		s.logger.Error("failed to list servers", slog.Any("error", err))
		return nil, err
	}

	ontimeMap, err := s.getServersOntime(ctx, servers)
	if err != nil {
		return nil, err
	}

	out := lo.Map(servers, func(server serverclient.ServerBrief, _ int) dto.ServerOntime {
		return dto.ServerOntime{ServerID: server.ID, DayStats: ontimeMap[server.ID]}
	})

	return out, nil
}

func (s *OntimeService) GetServersOntime(ctx context.Context, userID uuid.UUID, maxRecords int) (map[uint][]dto.DayStats, error) {

	perPage := maxRecords
	if perPage <= 0 {
		perPage = 10000
	}

	servers, err := s.serverClient.ListServers(ctx, userID, 1, perPage)
	if err != nil {
		s.logger.Error("failed to list servers for ontime", slog.String("user_id", userID.String()), slog.Any("error", err))
		return nil, err
	}

	return s.getServersOntime(ctx, servers)
}

func (s *OntimeService) GetServerWithOntime(ctx context.Context, serverID uint, userID uuid.UUID) (*dto.ServerOntime, error) {

	server, err := s.serverClient.GetServer(ctx, serverID, userID)
	if err != nil {
		return nil, err
	}

	ontimeMap, err := s.getServersOntime(ctx, []serverclient.ServerBrief{*server})
	if err != nil {
		return nil, err
	}

	return &dto.ServerOntime{ServerID: serverID, DayStats: ontimeMap[serverID]}, nil
}

func (s *OntimeService) getServersOntime(ctx context.Context, servers []serverclient.ServerBrief) (map[uint][]dto.DayStats, error) {

	dates := utils.Last30Days()
	items := make([]dto.BatchGetOntimeItem, 0, len(servers)*len(dates))
	serverDates := make(map[uint][]time.Time, len(servers))

	for _, sv := range servers {

		created := utils.TruncateDay(sv.CreatedAt)
		i := sort.Search(len(dates), func(i int) bool {
			return !dates[i].Before(created)
		})

		dates = dates[i:]

		datesIter := slices.Values(dates)
		newItems := it.Map(datesIter, func(d time.Time) dto.BatchGetOntimeItem {
			return dto.BatchGetOntimeItem{ServerID: sv.ID, Date: d}
		})

		items = slices.AppendSeq(items, newItems)
		serverDates[sv.ID] = dates
	}

	if len(items) == 0 {
		return make(map[uint][]dto.DayStats), nil
	}

	results, err := s.batcher.BatchGetOntime(ctx, items)
	if err != nil {
		s.logger.Error("failed to batch get ontime", slog.Any("error", err))
		return nil, err
	}

	lookup := buildOntimeLookup(results)

	out := make(map[uint][]dto.DayStats, len(servers))
	for _, sv := range servers {

		stats, ok := lookup[sv.ID]
		if !ok {
			stats = make(map[time.Time]dto.DayResult)
		}

		out[sv.ID] = lo.Map(serverDates[sv.ID], func(d time.Time, _ int) dto.DayStats {
			return dto.DayStats{Date: d, Result: stats[d]}
		})
	}

	return out, nil
}

func buildOntimeLookup(results []dto.ServerOntime) map[uint]map[time.Time]dto.DayResult {
	return lo.SliceToMap(results, func(result dto.ServerOntime) (uint, map[time.Time]dto.DayResult) {

		mp := lo.SliceToMap(result.DayStats, func(stat dto.DayStats) (time.Time, dto.DayResult) {
			return utils.TruncateDay(stat.Date), stat.Result
		})

		return result.ServerID, mp
	})
}
