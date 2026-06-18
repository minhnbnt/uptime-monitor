package token

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/jwt"
)

type TokenGenerator interface {
	GenerateAccessToken(user *domain.User) (string, error)
	GenerateRefreshToken(user *domain.User) (string, error)
}

type RevokedTokenRepository interface {
	Revoke(ctx context.Context, token *jwt.Token) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}
