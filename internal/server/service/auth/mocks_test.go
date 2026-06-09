package auth

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
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
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	return m.createFn(ctx, user)
}
func (m *mockUserRepo) FindByEmailOrUsername(ctx context.Context, login string) (*domain.User, error) {
	return m.findByEmailOrUsernameFn(ctx, login)
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
