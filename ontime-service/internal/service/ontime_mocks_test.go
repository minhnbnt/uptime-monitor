package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
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

type mockRangeRepo struct {
	batchGetOntimeRangeFn func(ctx context.Context, req []ontimerepo.BatchGetOntimeRangeRequest) ([]ontimerepo.ServerEvent, error)
}

func (m *mockRangeRepo) BatchGetOntimeRange(ctx context.Context, req []ontimerepo.BatchGetOntimeRangeRequest) ([]ontimerepo.ServerEvent, error) {
	return m.batchGetOntimeRangeFn(ctx, req)
}

var _ OntineRangeRepository = (*mockRangeRepo)(nil)

type mockOwnerRepo struct {
	listByUserIDFn           func(ctx context.Context, userID uuid.UUID, page, perPage int) ([]domain.ServerOwner, error)
	listByUserAndServerIDsFn func(ctx context.Context, userID uuid.UUID, serverIDs []uint) ([]domain.ServerOwner, error)
	getByServerIDFn          func(ctx context.Context, serverID uint) (*domain.ServerOwner, error)
	getByServerAndUserFn     func(ctx context.Context, serverID uint, userID uuid.UUID) (*domain.ServerOwner, error)
}

func (m *mockOwnerRepo) ListByUserID(ctx context.Context, userID uuid.UUID, page, perPage int) ([]domain.ServerOwner, error) {
	return m.listByUserIDFn(ctx, userID, page, perPage)
}

func (m *mockOwnerRepo) ListByUserAndServerIDs(ctx context.Context, userID uuid.UUID, serverIDs []uint) ([]domain.ServerOwner, error) {
	return m.listByUserAndServerIDsFn(ctx, userID, serverIDs)
}

func (m *mockOwnerRepo) GetByServerID(ctx context.Context, serverID uint) (*domain.ServerOwner, error) {
	return m.getByServerIDFn(ctx, serverID)
}

func (m *mockOwnerRepo) GetByServerAndUser(ctx context.Context, serverID uint, userID uuid.UUID) (*domain.ServerOwner, error) {
	return m.getByServerAndUserFn(ctx, serverID, userID)
}

var _ ServerOwnerRepository = (*mockOwnerRepo)(nil)
