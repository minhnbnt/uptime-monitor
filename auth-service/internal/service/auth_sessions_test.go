package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/logger"
)

func sessionRow(id uint, userID uint, jti string, scopes string, expires time.Time) domain.Session {
	parsed, err := uuid.Parse(jti)
	if err != nil {
		panic(err)
	}
	return domain.Session{
		Model:     gorm.Model{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		UserID:    userID,
		JTI:       parsed,
		Scopes:    scopes,
		ExpiresAt: expires,
	}
}

func TestAuthService_ListSessions(t *testing.T) {

	const userID = uint(7)
	now := time.Now()

	active := func(i int) domain.Session {
		scopes := "app"
		if i == 25 {
			scopes = "ping"
		}
		return sessionRow(
			uint(i+1),
			userID,
			fmt.Sprintf("0195f0b0-0000-7000-8000-%012d", i),
			scopes,
			now.Add(24*time.Hour),
		)
	}

	t.Run("filters expired marks current and paginates", func(t *testing.T) {

		rows := make([]domain.Session, 0, 46)
		for i := 0; i < 45; i++ {
			rows = append(rows, active(i))
		}
		expired := sessionRow(
			uint(len(rows)+1),
			userID,
			"0195f0b0-0000-7000-8000-000000000999",
			"app",
			now.Add(-time.Hour),
		)

		svc := &AuthService{
			logger: logger.NewMockLogger(),
			sessionRepository: &mockSessionRepo{
				findByUserFn: func(_ context.Context, id uint) ([]domain.Session, error) {
					if id != userID {
						t.Errorf("wrong user id: %d", id)
					}
					return append(rows, expired), nil
				},
			},
		}

		items, total, err := svc.ListSessions(context.Background(), userID, rows[25].JTI.String(), 2, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if total != 45 {
			t.Errorf("total = %d, want 45 (expired filtered)", total)
		}

		if len(items) != 20 {
			t.Errorf("len(items) = %d, want 20 (page 2 of per_page 20)", len(items))
		}

		if !items[5].Current {
			t.Error("items[5] should be the current session")
		}
		for i, item := range items {
			if i != 5 && item.Current {
				t.Errorf("items[%d] should not be current", i)
			}
		}

		if got := items[5].Scopes; len(got) != 1 || got[0] != "ping" {
			t.Errorf("items[5].scopes = %v, want [ping]", got)
		}
	})

	t.Run("empty list", func(t *testing.T) {

		svc := &AuthService{
			logger: logger.NewMockLogger(),
			sessionRepository: &mockSessionRepo{
				findByUserFn: func(_ context.Context, _ uint) ([]domain.Session, error) {
					return nil, nil
				},
			},
		}

		items, total, err := svc.ListSessions(context.Background(), userID, "", 1, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if total != 0 || len(items) != 0 {
			t.Errorf("want empty result, got total=%d items=%d", total, len(items))
		}
	})

	t.Run("page beyond range yields empty items with total kept", func(t *testing.T) {

		svc := &AuthService{
			logger: logger.NewMockLogger(),
			sessionRepository: &mockSessionRepo{
				findByUserFn: func(_ context.Context, _ uint) ([]domain.Session, error) {
					return []domain.Session{active(0)}, nil
				},
			},
		}

		items, total, err := svc.ListSessions(context.Background(), userID, "", 9, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if total != 1 || len(items) != 0 {
			t.Errorf("want total=1 items=0, got total=%d items=%d", total, len(items))
		}
	})

	t.Run("repo error maps to internal", func(t *testing.T) {

		svc := &AuthService{
			logger: logger.NewMockLogger(),
			sessionRepository: &mockSessionRepo{
				findByUserFn: func(_ context.Context, _ uint) ([]domain.Session, error) {
					return nil, errors.New("db down")
				},
			},
		}

		_, _, err := svc.ListSessions(context.Background(), userID, "", 1, 20)
		if !errors.Is(err, apperrors.ErrInternal) {
			t.Errorf("err = %v, want ErrInternal", err)
		}
	})
}

func TestAuthService_RevokeSession(t *testing.T) {

	const userID = uint(7)

	t.Run("found revokes successfully", func(t *testing.T) {

		var gotUserID uint
		var gotJTI string

		svc := &AuthService{
			logger: logger.NewMockLogger(),
			sessionRepository: &mockSessionRepo{
				deleteByJTIAndUserFn: func(_ context.Context, id uint, jti string) (bool, error) {
					gotUserID, gotJTI = id, jti
					return true, nil
				},
			},
		}

		err := svc.RevokeSession(context.Background(), userID, "0195f0b0-0000-7000-8000-000000000001")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if gotUserID != userID {
			t.Errorf("user id = %d, want %d", gotUserID, userID)
		}
		if gotJTI != "0195f0b0-0000-7000-8000-000000000001" {
			t.Errorf("jti = %q", gotJTI)
		}
	})

	t.Run("not found or foreign session", func(t *testing.T) {

		svc := &AuthService{
			logger: logger.NewMockLogger(),
			sessionRepository: &mockSessionRepo{
				deleteByJTIAndUserFn: func(_ context.Context, _ uint, _ string) (bool, error) {
					return false, nil
				},
			},
		}

		err := svc.RevokeSession(context.Background(), userID, "0195f0b0-0000-7000-8000-000000000002")
		if !errors.Is(err, apperrors.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("repo error maps to internal", func(t *testing.T) {

		svc := &AuthService{
			logger: logger.NewMockLogger(),
			sessionRepository: &mockSessionRepo{
				deleteByJTIAndUserFn: func(_ context.Context, _ uint, _ string) (bool, error) {
					return false, errors.New("db down")
				},
			},
		}

		err := svc.RevokeSession(context.Background(), userID, "0195f0b0-0000-7000-8000-000000000003")
		if !errors.Is(err, apperrors.ErrInternal) {
			t.Errorf("err = %v, want ErrInternal", err)
		}
	})
}
