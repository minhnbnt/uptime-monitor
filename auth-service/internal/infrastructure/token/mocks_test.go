package token

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/domain"
)

type mockSessionRepo struct {
	getByJTIFn func(ctx context.Context, jti string) (*domain.Session, error)
}

func (m *mockSessionRepo) GetByJTI(ctx context.Context, jti string) (*domain.Session, error) {
	if m.getByJTIFn == nil {
		return &domain.Session{
			UserID:    42,
			JTI:       uuid.MustParse(jti),
			Scopes:    "app",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}, nil
	}
	return m.getByJTIFn(ctx, jti)
}
