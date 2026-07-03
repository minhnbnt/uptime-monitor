package handler

import (
	"context"

	ontimedto "github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
)

type mockOntimeService struct {
	listServersWithOntimeFn func(ctx context.Context, createdByID uint, page, perPage int) ([]ontimedto.ServerWithOntime, int64, error)
	getServerWithOntimeFn   func(ctx context.Context, serverID uint, userID uint) (*ontimedto.ServerWithOntime, error)
}

func (m *mockOntimeService) ListServersWithOntime(ctx context.Context, createdByID uint, page, perPage int) ([]ontimedto.ServerWithOntime, int64, error) {
	if m.listServersWithOntimeFn == nil {
		return nil, 0, nil
	}
	return m.listServersWithOntimeFn(ctx, createdByID, page, perPage)
}

func (m *mockOntimeService) GetServerWithOntime(ctx context.Context, serverID uint, userID uint) (*ontimedto.ServerWithOntime, error) {
	return m.getServerWithOntimeFn(ctx, serverID, userID)
}
