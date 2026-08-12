package handler

import (
	"context"

	"github.com/google/uuid"

	ontimedto "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
)

type mockOntimeService struct {
	getServerWithOntimeFn  func(ctx context.Context, serverID uint, userID uuid.UUID) (*ontimedto.ServerOntime, error)
	getServersWithOntimeFn func(ctx context.Context, userID uuid.UUID, ids []uint) ([]ontimedto.ServerOntime, error)
}

func (m *mockOntimeService) GetServerWithOntime(ctx context.Context, serverID uint, userID uuid.UUID) (*ontimedto.ServerOntime, error) {
	return m.getServerWithOntimeFn(ctx, serverID, userID)
}

func (m *mockOntimeService) GetServersWithOntime(ctx context.Context, userID uuid.UUID, ids []uint) ([]ontimedto.ServerOntime, error) {
	return m.getServersWithOntimeFn(ctx, userID, ids)
}
