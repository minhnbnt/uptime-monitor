package token

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/domain"
)

type SessionRepository interface {
	GetByJTI(ctx context.Context, jti string) (*domain.Session, error)
}

type Generator interface {
	GenerateAccessToken(user *domain.User, scopes []string, sessionID string) (string, error)
	GenerateRefreshToken(user *domain.User) (string, string, error)
}
