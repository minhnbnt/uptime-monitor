package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/repository/ontime"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
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

type mockServerRepo struct {
	listFn           func(ctx context.Context, createdByID uint, limit, offset int) ([]domain.Server, error)
	countFn          func(ctx context.Context, createdByID uint) (int64, error)
	createFn         func(ctx context.Context, s *domain.Server) error
	getByIDFn        func(ctx context.Context, id uint) (*domain.Server, error)
	updateFn         func(ctx context.Context, s *domain.Server) error
	deleteFn         func(ctx context.Context, id uint) error
	batchGetOntimeFn func(ctx context.Context, req []serverrepo.BatchGetOntimeRequest) ([]serverrepo.RawEvent, error)
}

func (m *mockServerRepo) List(ctx context.Context, createdByID uint, limit, offset int) ([]domain.Server, error) {
	return m.listFn(ctx, createdByID, limit, offset)
}
func (m *mockServerRepo) Count(ctx context.Context, createdByID uint) (int64, error) {
	return m.countFn(ctx, createdByID)
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
func (m *mockServerRepo) BatchGetOntime(ctx context.Context, req []serverrepo.BatchGetOntimeRequest) ([]serverrepo.RawEvent, error) {
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
	parseFn    func(token string) (*jwtutil.Token, error)
}

func (m *mockTokenParser) NewToken(issuer string, otherClaims map[string]any) (string, error) {
	return m.newTokenFn(issuer, otherClaims)
}
func (m *mockTokenParser) Validate(token string) (issuer string, err error) {
	return m.validateFn(token)
}
func (m *mockTokenParser) Parse(token string) (*jwtutil.Token, error) {
	return m.parseFn(token)
}

type mockOntimeCacheRepo struct {
	mGetFn func(ctx context.Context, keys []ontimerepo.OntimeCacheKey) (map[ontimerepo.OntimeCacheKey]float64, error)
	mSetFn func(ctx context.Context, items map[ontimerepo.OntimeCacheKey]float64) error
}

func (m *mockOntimeCacheRepo) MGet(ctx context.Context, keys []ontimerepo.OntimeCacheKey) (map[ontimerepo.OntimeCacheKey]float64, error) {
	if m.mGetFn == nil {
		return nil, nil
	}
	return m.mGetFn(ctx, keys)
}

func (m *mockOntimeCacheRepo) MSet(ctx context.Context, items map[ontimerepo.OntimeCacheKey]float64) error {
	if m.mSetFn == nil {
		return nil
	}
	return m.mSetFn(ctx, items)
}

type mockLogger struct {
	infoCalled bool
	warnCalled bool
	lastMsg    string
}

func (m *mockLogger) Info(msg string, fields ...logger.Field) {
	m.infoCalled = true
	m.lastMsg = msg
}
func (m *mockLogger) Warn(msg string, fields ...logger.Field) {
	m.warnCalled = true
	m.lastMsg = msg
}
func (m *mockLogger) Error(msg string, fields ...logger.Field)  {}
func (m *mockLogger) Debug(msg string, fields ...logger.Field)  {}
func (m *mockLogger) Fatal(msg string, fields ...logger.Field)  {}
func (m *mockLogger) With(fields ...logger.Field) logger.Logger { return m }
