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
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/serverclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
)

type ServerClient interface {
	ListServers(ctx context.Context, userID uint, page, perPage int) ([]serverclient.ServerBrief, error)
	GetServer(ctx context.Context, serverID uint, userID uint) (*serverclient.ServerBrief, error)
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

func (s *OntimeService) ListServersWithOntime(ctx context.Context, userID uint, page, perPage int) ([]dto.ServerOntime, error) {

	servers, err := s.serverClient.ListServers(ctx, userID, page, perPage)
	if err != nil {
		s.logger.Error("failed to list servers", slog.Any("error", err))
		return nil, err
	}

	ontimeMap, err := s.getServersOntime(ctx, servers)
	if err != nil {
		return nil, err
	}

	out := make([]dto.ServerOntime, 0, len(servers))
	for _, sv := range servers {
		out = append(out, dto.ServerOntime{
			ServerID:    sv.ID,
			OntimeStats: ontimeMap[sv.ID],
		})
	}

	return out, nil
}

func (s *OntimeService) GetServerWithOntime(ctx context.Context, serverID, userID uint) (*dto.ServerOntime, error) {

	server, err := s.serverClient.GetServer(ctx, serverID, userID)
	if err != nil {
		return nil, err
	}

	ontimeMap, err := s.getServersOntime(ctx, []serverclient.ServerBrief{*server})
	if err != nil {
		return nil, err
	}

	return &dto.ServerOntime{
		ServerID:    serverID,
		OntimeStats: ontimeMap[serverID],
	}, nil
}

func (s *OntimeService) getServersOntime(ctx context.Context, servers []serverclient.ServerBrief) (map[uint][]dto.OntimeStats, error) {

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
		s.logger.Error("failed to batch get ontime", slog.Any("error", err))
		return nil, err
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
