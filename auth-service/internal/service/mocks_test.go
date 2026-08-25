package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/domain"
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
	generateAccessTokenFn  func(user *domain.User, scopes []string, sessionID string) (string, error)
	generateRefreshTokenFn func(user *domain.User) (string, string, error)
}

func (m *mockTokenGenerator) GenerateAccessToken(user *domain.User, scopes []string, sessionID string) (string, error) {
	return m.generateAccessTokenFn(user, scopes, sessionID)
}
func (m *mockTokenGenerator) GenerateRefreshToken(user *domain.User) (string, string, error) {
	return m.generateRefreshTokenFn(user)
}

type mockSessionRepo struct {
	createFn             func(ctx context.Context, session *domain.Session) error
	getByJTIFn           func(ctx context.Context, jti string) (*domain.Session, error)
	deleteByJTIFn        func(ctx context.Context, jti string) error
	deleteByJTIAndUserFn func(ctx context.Context, userID uint, jti string) (bool, error)
	findByUserFn         func(ctx context.Context, userID uint) ([]domain.Session, error)
}

func (m *mockSessionRepo) Create(ctx context.Context, session *domain.Session) error {
	if m.createFn == nil {
		return nil
	}
	return m.createFn(ctx, session)
}

func (m *mockSessionRepo) GetByJTI(ctx context.Context, jti string) (*domain.Session, error) {
	if m.getByJTIFn == nil {
		return &domain.Session{
			UserID:    42,
			JTI:       uuid.MustParse(jti),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}, nil
	}
	return m.getByJTIFn(ctx, jti)
}

func (m *mockSessionRepo) DeleteByJTI(ctx context.Context, jti string) error {
	if m.deleteByJTIFn == nil {
		return nil
	}
	return m.deleteByJTIFn(ctx, jti)
}

func (m *mockSessionRepo) FindByUser(ctx context.Context, userID uint) ([]domain.Session, error) {
	if m.findByUserFn == nil {
		return nil, nil
	}
	return m.findByUserFn(ctx, userID)
}

func (m *mockSessionRepo) DeleteByJTIAndUser(ctx context.Context, userID uint, jti string) (bool, error) {
	if m.deleteByJTIAndUserFn == nil {
		return false, nil
	}
	return m.deleteByJTIAndUserFn(ctx, userID, jti)
}
