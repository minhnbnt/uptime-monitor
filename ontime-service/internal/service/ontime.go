package service

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/samber/lo/it"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
)

// OwnerRepository reads ownership from the local server_owners table
// (replicated via the ownership Redis Stream consumer), so ontime lookups
// never need to call out to the server-service over gRPC.
type OwnerRepository interface {
	ListByUserID(ctx context.Context, userID uuid.UUID, page, perPage int) ([]domain.ServerOwner, error)
	ListByUserAndServerIDs(ctx context.Context, userID uuid.UUID, serverIDs []uint) ([]domain.ServerOwner, error)
	GetByServerID(ctx context.Context, serverID uint) (*domain.ServerOwner, error)
	GetByServerAndUser(ctx context.Context, serverID uint, userID uuid.UUID) (*domain.ServerOwner, error)
}

type OntimeService struct {
	ownerRepo OwnerRepository
	logger    *slog.Logger
	batcher   *Batcher
}

func NewOntimeService(ownerRepo OwnerRepository, b *Batcher, l *slog.Logger) *OntimeService {
	return &OntimeService{
		ownerRepo: ownerRepo,
		batcher:   b,
		logger:    l,
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

func (s *OntimeService) GetServersOntime(ctx context.Context, userID uuid.UUID, maxRecords int) (map[uint][]dto.DayStats, error) {

	perPage := maxRecords
	if perPage <= 0 {
		perPage = 10000
	}

	servers, err := s.ownerRepo.ListByUserID(ctx, userID, 1, perPage)
	if err != nil {
		s.logger.Error("failed to list servers for ontime", slog.String("user_id", userID.String()), slog.Any("error", err))
		return nil, err
	}

	return s.getServersOntime(ctx, servers)
}

func (s *OntimeService) GetServerWithOntime(ctx context.Context, serverID uint, userID uuid.UUID) (*dto.ServerOntime, error) {

	owner, err := s.ownerRepo.GetByServerAndUser(ctx, serverID, userID)
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.ErrForbidden
	}
	if err != nil {
		return nil, err
	}

	ontimeMap, err := s.getServersOntime(ctx, []domain.ServerOwner{*owner})
	if err != nil {
		return nil, err
	}

	return &dto.ServerOntime{ServerID: serverID, DayStats: ontimeMap[serverID]}, nil
}

// batchServerCap is the maximum number of server IDs accepted by a single
// batch ontime request. Beyond this the request is rejected with 400.
const batchServerCap = 100

// GetServersWithOntime returns ontime stats for the requested server IDs that
// the caller actually owns; unowned (or unknown) IDs are silently dropped from
// the result. Requests with more than batchServerCap IDs are rejected with
// ErrBadRequest.
func (s *OntimeService) GetServersWithOntime(ctx context.Context, userID uuid.UUID, ids []uint) ([]dto.ServerOntime, error) {

	if len(ids) > batchServerCap {
		return nil, apperrors.ErrBadRequest
	}

	owned, err := s.ownerRepo.ListByUserAndServerIDs(ctx, userID, ids)
	if err != nil {
		s.logger.Error("failed to check server ownership", slog.String("user_id", userID.String()), slog.Any("error", err))
		return nil, err
	}

	ontimeMap, err := s.getServersOntime(ctx, owned)
	if err != nil {
		return nil, err
	}

	out := lo.Map(owned, func(owner domain.ServerOwner, _ int) dto.ServerOntime {
		return dto.ServerOntime{
			DayStats: ontimeMap[owner.ServerID],
			ServerID: owner.ServerID,
		}
	})

	return out, nil
}

func (s *OntimeService) getServersOntime(ctx context.Context, servers []domain.ServerOwner) (map[uint][]dto.DayStats, error) {

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
			return dto.BatchGetOntimeItem{ServerID: sv.ServerID, Date: d}
		})

		items = slices.AppendSeq(items, newItems)
		serverDates[sv.ServerID] = dates
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

		stats, ok := lookup[sv.ServerID]
		if !ok {
			stats = make(map[time.Time]dto.DayResult)
		}

		out[sv.ServerID] = lo.Map(serverDates[sv.ServerID], func(d time.Time, _ int) dto.DayStats {
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
