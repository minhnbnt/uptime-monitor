package ontime

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/features/ontime/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/features/ontime/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/serverclient"
)

type mockOntineRepo struct {
	batchGetOntimeFn func(ctx context.Context, req []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.RawEvent, error)
}

func (m *mockOntineRepo) BatchGetOntime(ctx context.Context, req []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.RawEvent, error) {
	return m.batchGetOntimeFn(ctx, req)
}

var _ OntineRepository = (*mockOntineRepo)(nil)

type mockOntimeCacheRepo struct {
	mGetFn func(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error)
	mSetFn func(ctx context.Context, items map[dto.BatchGetOntimeItem]float64) error
}

func (m *mockOntimeCacheRepo) MGet(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error) {
	if m.mGetFn != nil {
		return m.mGetFn(ctx, keys)
	}
	return make(map[dto.BatchGetOntimeItem]float64), nil
}

func (m *mockOntimeCacheRepo) MSet(ctx context.Context, items map[dto.BatchGetOntimeItem]float64) error {
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
