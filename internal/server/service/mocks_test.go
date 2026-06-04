package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	repo "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/repository"
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

type mockServerRepo struct {
	listFn           func(ctx context.Context, limit, offset int) ([]domain.Server, error)
	countFn          func(ctx context.Context) (int64, error)
	createFn         func(ctx context.Context, s *domain.Server) error
	getByIDFn        func(ctx context.Context, id uint) (*domain.Server, error)
	updateFn         func(ctx context.Context, s *domain.Server) error
	deleteFn         func(ctx context.Context, id uint) error
	batchGetOntimeFn func(ctx context.Context, req []repo.BatchGetOntimeRequest) ([]repo.RawEvent, error)
}

func (m *mockServerRepo) List(ctx context.Context, limit, offset int) ([]domain.Server, error) {
	return m.listFn(ctx, limit, offset)
}
func (m *mockServerRepo) Count(ctx context.Context) (int64, error) {
	return m.countFn(ctx)
}
func (m *mockServerRepo) Create(ctx context.Context, s *domain.Server) error {
	return m.createFn(ctx, s)
}
func (m *mockServerRepo) GetByID(ctx context.Context, id uint) (*domain.Server, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockServerRepo) Update(ctx context.Context, s *domain.Server) error {
	return m.updateFn(ctx, s)
}
func (m *mockServerRepo) Delete(ctx context.Context, id uint) error {
	return m.deleteFn(ctx, id)
}
func (m *mockServerRepo) BatchGetOntime(ctx context.Context, req []repo.BatchGetOntimeRequest) ([]repo.RawEvent, error) {
	return m.batchGetOntimeFn(ctx, req)
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

type mockEndpointRepo struct {
	upsertEndpointFn func(ctx context.Context, endpoint domain.Endpoint) error
}

func (m *mockEndpointRepo) UpsertEndpoint(ctx context.Context, endpoint domain.Endpoint) error {
	return m.upsertEndpointFn(ctx, endpoint)
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

type mockTokenParser struct {
	newTokenFn func(issuer string, otherClaims map[string]any) (string, error)
	validateFn func(token string) (issuer string, err error)
}

func (m *mockTokenParser) NewToken(issuer string, otherClaims map[string]any) (string, error) {
	return m.newTokenFn(issuer, otherClaims)
}
func (m *mockTokenParser) Validate(token string) (issuer string, err error) {
	return m.validateFn(token)
}
