package service

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/serverclient"
)

type mockOntimeRepo struct {
	batchGetUptimeFn func(ctx context.Context, req []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.UptimeRow, error)
}

func (m *mockOntimeRepo) BatchGetUptime(ctx context.Context, req []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.UptimeRow, error) {
	if m.batchGetUptimeFn == nil {
		return nil, nil
	}
	return m.batchGetUptimeFn(ctx, req)
}

var _ OntimeRepository = (*mockOntimeRepo)(nil)

type mockOntimeCacheRepo struct {
	mGetFn func(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]dto.DayResult, error)
	mSetFn func(ctx context.Context, items map[dto.BatchGetOntimeItem]dto.DayResult) error
}

func (m *mockOntimeCacheRepo) MGet(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]dto.DayResult, error) {
	if m.mGetFn == nil {
		return make(map[dto.BatchGetOntimeItem]dto.DayResult), nil
	}
	return m.mGetFn(ctx, keys)
}

func (m *mockOntimeCacheRepo) MSet(ctx context.Context, items map[dto.BatchGetOntimeItem]dto.DayResult) error {
	if m.mSetFn == nil {
		return nil
	}
	return m.mSetFn(ctx, items)
}

var _ OntimeCacheRepository = (*mockOntimeCacheRepo)(nil)

type mockServerClient struct {
	listServersFn func(ctx context.Context, userID uint, page, perPage int) ([]serverclient.ServerBrief, error)
	getServerFn   func(ctx context.Context, serverID uint, userID uint) (*serverclient.ServerBrief, error)
}

func (m *mockServerClient) ListServers(ctx context.Context, userID uint, page, perPage int) ([]serverclient.ServerBrief, error) {
	return m.listServersFn(ctx, userID, page, perPage)
}

func (m *mockServerClient) GetServer(ctx context.Context, serverID uint, userID uint) (*serverclient.ServerBrief, error) {
	return m.getServerFn(ctx, serverID, userID)
}

var _ ServerClient = (*mockServerClient)(nil)

type mockServerOwnerRepo struct {
	ownedServersFn func(ctx context.Context, userID uint, serverIDs []uint) ([]ontimerepo.OwnedServer, error)
}

func (m *mockServerOwnerRepo) GetOwnedServers(ctx context.Context, userID uint, serverIDs []uint) ([]ontimerepo.OwnedServer, error) {
	if m.ownedServersFn == nil {
		return nil, nil
	}
	return m.ownedServersFn(ctx, userID, serverIDs)
}

func (m *mockServerOwnerRepo) GetOwnedServerIDs(_ context.Context, _ uint, _ []uint) ([]uint, error) {
	return nil, nil
}

var _ ServerOwnerRepository = (*mockServerOwnerRepo)(nil)
