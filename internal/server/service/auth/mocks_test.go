package auth

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
)

func gormModel(id uint, t time.Time) gorm.Model {
	return gorm.Model{ID: id, CreatedAt: t, UpdatedAt: t}
}

func domainUser(id uint, email, username string) domain.User {
	return domain.User{
		Model:    gormModel(id, time.Now()),
		Email:    email,
		Username: username,
		Password: "hashed-password",
		Name:     "Test User",
	}
}

type mockUserRepo struct {
	createFn                func(ctx context.Context, user *domain.User) error
	findByEmailOrUsernameFn func(ctx context.Context, login string) (*domain.User, error)
	findByIDFn              func(ctx context.Context, id uint) (*domain.User, error)
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	return m.createFn(ctx, user)
}
func (m *mockUserRepo) FindByEmailOrUsername(ctx context.Context, login string) (*domain.User, error) {
	return m.findByEmailOrUsernameFn(ctx, login)
}
func (m *mockUserRepo) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	return m.findByIDFn(ctx, id)
}

type mockPasswordEncoder struct {
	encodeFn func(password string) (string, error)
	verifyFn func(password, encodedHash string) (bool, error)
}

func (m *mockPasswordEncoder) Encode(password string) (string, error) {
	return m.encodeFn(password)
}
func (m *mockPasswordEncoder) Verify(password, encodedHash string) (bool, error) {
	return m.verifyFn(password, encodedHash)
}

type mockTokenGenerator struct {
	generateAccessTokenFn  func(user *domain.User) (string, error)
	generateRefreshTokenFn func(user *domain.User) (string, error)
}

func (m *mockTokenGenerator) GenerateAccessToken(user *domain.User) (string, error) {
	return m.generateAccessTokenFn(user)
}
func (m *mockTokenGenerator) GenerateRefreshToken(user *domain.User) (string, error) {
	return m.generateRefreshTokenFn(user)
}

type mockRevokedTokenRepo struct {
	revokeFn    func(ctx context.Context, token *jwtutil.Token) error
	isRevokedFn func(ctx context.Context, jti string) (bool, error)
}

func (m *mockRevokedTokenRepo) Revoke(ctx context.Context, token *jwtutil.Token) error {
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
