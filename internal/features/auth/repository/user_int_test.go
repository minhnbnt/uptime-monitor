package repository

import (
	"errors"
	"testing"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
)

func userRepo(tb testing.TB) *UserRepository {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	return NewUserRepository(testDB)
}

func cleanUsers(tb testing.TB) {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	if err := testDB.Where("1 = 1").Delete(&domain.User{}).Error; err != nil {
		tb.Fatalf("clean users: %v", err)
	}
}

func TestIntegration_CreateUser_Success(t *testing.T) {
	cleanUsers(t)

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
	cleanUsers(t)

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
	cleanUsers(t)

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
