package handler

import (
	"context"

	"github.com/google/uuid"

	ontimedto "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
)

type mockOntimeService struct {
	listServersWithOntimeFn func(ctx context.Context, createdByID uuid.UUID, page, perPage int) ([]ontimedto.ServerOntime, error)
	getServerWithOntimeFn   func(ctx context.Context, serverID uint, userID uuid.UUID) (*ontimedto.ServerOntime, error)
}

func (m *mockOntimeService) ListServersWithOntime(ctx context.Context, createdByID uuid.UUID, page, perPage int) ([]ontimedto.ServerOntime, error) {
	if m.listServersWithOntimeFn == nil {
		return nil, nil
	}
	return m.listServersWithOntimeFn(ctx, createdByID, page, perPage)
}

func (m *mockOntimeService) GetServerWithOntime(ctx context.Context, serverID uint, userID uuid.UUID) (*ontimedto.ServerOntime, error) {
	return m.getServerWithOntimeFn(ctx, serverID, userID)
}
