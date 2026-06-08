package service

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
	repo "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/repository"
)

type ServerRepository interface {
	Count(ctx context.Context, createdByID uint) (int64, error)
	List(ctx context.Context, createdByID uint, limit, offset int) ([]domain.Server, error)
	Create(ctx context.Context, s *domain.Server) error
	GetByID(ctx context.Context, id uint) (*domain.Server, error)
	Update(ctx context.Context, s *domain.Server) error
	Delete(ctx context.Context, id uint) error
	BatchGetOntime(ctx context.Context, req []repo.BatchGetOntimeRequest) ([]repo.RawEvent, error)
}

type EndpointRepository interface {
	UpsertEndpoint(ctx context.Context, endpoint domain.Endpoint) error
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmailOrUsername(ctx context.Context, login string) (*domain.User, error)
}

type OntimeCacheRepository interface {
	MGet(ctx context.Context, keys []repo.OntimeCacheKey) (map[repo.OntimeCacheKey]float64, error)
	MSet(ctx context.Context, items map[repo.OntimeCacheKey]float64) error
}

type PasswordEncoder interface {
	Encode(password string) (string, error)
	Verify(password, encodedHash string) (bool, error)
}

type TokenParser interface {
	NewToken(issuer string, otherClaims map[string]any) (string, error)
	Validate(token string) (issuer string, err error)
	Parse(token string) (*jwtutil.Token, error)
}
