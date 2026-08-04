package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/serverclient"
)

type mockOntineRepo struct {
	batchGetOntimeFn func(ctx context.Context, req []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.ServerEvent, error)
}

func (m *mockOntineRepo) BatchGetOntime(ctx context.Context, req []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.ServerEvent, error) {
	return m.batchGetOntimeFn(ctx, req)
}

var _ OntineRepository = (*mockOntineRepo)(nil)

type mockOntimeCacheRepo struct {
	mGetFn func(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]dto.DayResult, error)
	mSetFn func(ctx context.Context, items map[dto.BatchGetOntimeItem]dto.DayResult) error
}

func (m *mockOntimeCacheRepo) MGet(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]dto.DayResult, error) {
	if m.mGetFn != nil {
		return m.mGetFn(ctx, keys)
	}
	return make(map[dto.BatchGetOntimeItem]dto.DayResult), nil
}

func (m *mockOntimeCacheRepo) MSet(ctx context.Context, items map[dto.BatchGetOntimeItem]dto.DayResult) error {
	if m.mSetFn == nil {
		return nil
	}
	return m.mSetFn(ctx, items)
}

var _ OntimeCacheRepository = (*mockOntimeCacheRepo)(nil)

type mockServerClient struct {
	listServersFn func(ctx context.Context, userID uuid.UUID, page, perPage int) ([]serverclient.ServerBrief, error)
	getServerFn   func(ctx context.Context, serverID uint, userID uuid.UUID) (*serverclient.ServerBrief, error)
}

func (m *mockServerClient) ListServers(ctx context.Context, userID uuid.UUID, page, perPage int) ([]serverclient.ServerBrief, error) {
	return m.listServersFn(ctx, userID, page, perPage)
}

func (m *mockServerClient) GetServer(ctx context.Context, serverID uint, userID uuid.UUID) (*serverclient.ServerBrief, error) {
	return m.getServerFn(ctx, serverID, userID)
}

var _ ServerClient = (*mockServerClient)(nil)

type mockRangeRepo struct {
	batchGetOntimeRangeFn func(ctx context.Context, req []ontimerepo.BatchGetOntimeRangeRequest) ([]ontimerepo.ServerEvent, error)
}

func (m *mockRangeRepo) BatchGetOntimeRange(ctx context.Context, req []ontimerepo.BatchGetOntimeRangeRequest) ([]ontimerepo.ServerEvent, error) {
	return m.batchGetOntimeRangeFn(ctx, req)
}

var _ OntineRangeRepository = (*mockRangeRepo)(nil)
