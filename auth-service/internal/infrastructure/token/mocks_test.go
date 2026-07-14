package token

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/jwt"
)

type mockRevokedTokenRepo struct {
	revokeFn    func(ctx context.Context, token *jwt.Token) error
	isRevokedFn func(ctx context.Context, jti string) (bool, error)
}

func (m *mockRevokedTokenRepo) Revoke(ctx context.Context, token *jwt.Token) error {
	if m.revokeFn == nil {
		return nil
	}
	return m.revokeFn(ctx, token)
}

func (m *mockRevokedTokenRepo) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if m.isRevokedFn == nil {
		return false, nil
	}
	return m.isRevokedFn(ctx, jti)
}
