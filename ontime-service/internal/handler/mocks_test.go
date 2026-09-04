package handler

import (
	"context"
	"time"

	ontimedto "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
)

type mockOntimeService struct {
	listServersWithOntimeFn func(ctx context.Context, createdByID uint, page, perPage int) ([]ontimedto.ServerOntime, error)
	getServerWithOntimeFn   func(ctx context.Context, serverID uint, userID uint) (*ontimedto.ServerOntime, error)
	getServersWithOntimeFn  func(ctx context.Context, userID uint, ids []uint, loc *time.Location) ([]ontimedto.ServerOntime, error)
}

func (m *mockOntimeService) ListServersWithOntime(ctx context.Context, createdByID uint, page, perPage int) ([]ontimedto.ServerOntime, error) {
	if m.listServersWithOntimeFn == nil {
		return nil, nil
	}
	return m.listServersWithOntimeFn(ctx, createdByID, page, perPage)
}

func (m *mockOntimeService) GetServerWithOntime(ctx context.Context, serverID uint, userID uint) (*ontimedto.ServerOntime, error) {
	return m.getServerWithOntimeFn(ctx, serverID, userID)
}

func (m *mockOntimeService) GetServersWithOntime(ctx context.Context, userID uint, ids []uint, loc *time.Location) ([]ontimedto.ServerOntime, error) {
	if m.getServersWithOntimeFn == nil {
		return nil, nil
	}
	return m.getServersWithOntimeFn(ctx, userID, ids, loc)
}

type mockOntimeRangeService struct {
	calculateUptimeFn func(ctx context.Context, in ontimedto.CalculateUptimeInput) (*ontimedto.UptimeResponse, error)
}

func (m *mockOntimeRangeService) CalculateUptime(ctx context.Context, in ontimedto.CalculateUptimeInput) (*ontimedto.UptimeResponse, error) {
	if m.calculateUptimeFn == nil {
		return nil, nil
	}
	return m.calculateUptimeFn(ctx, in)
}
