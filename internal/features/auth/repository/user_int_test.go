package repository

import (
	"errors"
	"testing"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

func userRepo(tb testing.TB) *UserRepository {
	tb.Helper()
	testcontainers.SkipIfShort(tb)
	db := testcontainers.CreateTestDB(tb, testDSN)
	return NewUserRepository(db)
}

func TestIntegration_CreateUser_Success(t *testing.T) {
	testcontainers.SkipIfShort(t)

	repo := userRepo(t)

	user := &domain.User{
		Email:    "success@test.com",
		Username: "success_user",
		Password: "hashed-pass",
		Name:     "Success",
	}

	err := repo.Create(t.Context(), user)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected user ID to be set after creation")
	}
}

func TestIntegration_CreateUser_DuplicateEmail(t *testing.T) {
	testcontainers.SkipIfShort(t)

	repo := userRepo(t)

	user1 := &domain.User{
		Email:    "dup@test.com",
		Username: "user_one",
		Password: "hashed-pass",
		Name:     "User One",
	}
	if err := repo.Create(t.Context(), user1); err != nil {
		t.Fatalf("first Create error: %v", err)
	}

	user2 := &domain.User{
		Email:    "dup@test.com",
		Username: "user_two",
		Password: "hashed-pass",
		Name:     "User Two",
	}
	err := repo.Create(t.Context(), user2)
	if !errors.Is(err, apperrors.ErrEmailOrUsernameTaken) {
		t.Errorf("got %v, want apperrors.ErrEmailOrUsernameTaken", err)
	}
}

func TestIntegration_CreateUser_DuplicateUsername(t *testing.T) {
	testcontainers.SkipIfShort(t)

	repo := userRepo(t)

	user1 := &domain.User{
		Email:    "first@test.com",
		Username: "dup_user",
		Password: "hashed-pass",
		Name:     "First",
	}
	if err := repo.Create(t.Context(), user1); err != nil {
		t.Fatalf("first Create error: %v", err)
	}

	user2 := &domain.User{
		Email:    "second@test.com",
		Username: "dup_user",
		Password: "hashed-pass",
		Name:     "Second",
	}
	err := repo.Create(t.Context(), user2)
	if !errors.Is(err, apperrors.ErrEmailOrUsernameTaken) {
		t.Errorf("got %v, want apperrors.ErrEmailOrUsernameTaken", err)
	}
}
