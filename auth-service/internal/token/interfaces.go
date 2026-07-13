package token

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/jwt"
)

type TokenGenerator interface {
	GenerateAccessToken(user *domain.User) (string, error)
	GenerateRefreshToken(user *domain.User) (string, error)
}

type RevokedTokenRepository interface {
	Revoke(ctx context.Context, token *jwt.Token) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}
