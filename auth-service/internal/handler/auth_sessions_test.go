package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/token"
)

func withTokenInfo(ctx context.Context, info *token.AccessTokenInfo) context.Context {
	return context.WithValue(ctx, tokenInfoKey{}, info)
}

func TestAuthHandler_ListSessions(t *testing.T) {

	sessionID := uuid.MustParse("0195f0b0-0000-7000-8000-000000000001")

	t.Run("happy path maps service result and meta", func(t *testing.T) {

		var gotUserID uint
		var gotSID string
		var gotPage, gotPerPage int

		h := &AuthHandler{
			sessionService: &mockSessionService{
				listSessionsFn: func(_ context.Context, userID uint, sid string, page, perPage int) ([]dto.SessionInfo, int, error) {
					gotUserID, gotSID, gotPage, gotPerPage = userID, sid, page, perPage
					return []dto.SessionInfo{
						{
							ID:      sessionID.String(),
							Scopes:  []string{"app"},
							Current: true,
						},
					}, 1, nil
				},
			},
		}

		ctx := withTokenInfo(t.Context(), &token.AccessTokenInfo{
			UserID: 7,
			Scopes: []string{"app"},
			SID:    sessionID.String(),
		})

		resp, err := h.ListSessions(ctx, api.ListSessionsParams{
			Page:    api.NewOptInt(2),
			PerPage: api.NewOptInt(50),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if gotUserID != 7 || gotSID != sessionID.String() || gotPage != 2 || gotPerPage != 50 {
			t.Errorf("service args = (%d, %q, %d, %d)", gotUserID, gotSID, gotPage, gotPerPage)
		}

		if len(resp.Data) != 1 || !resp.Data[0].Current {
			t.Errorf("unexpected data: %+v", resp.Data)
		}

		if resp.Meta.Page.Value != 2 || resp.Meta.PerPage.Value != 50 || resp.Meta.Total.Value != 1 {
			t.Errorf("meta = %+v", resp.Meta)
		}
	})

	t.Run("missing scope is forbidden", func(t *testing.T) {

		h := &AuthHandler{}

		ctx := withTokenInfo(t.Context(), &token.AccessTokenInfo{
			UserID: 7,
			Scopes: []string{"ping"},
		})

		_, err := h.ListSessions(ctx, api.ListSessionsParams{})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusForbidden {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusForbidden)
		}
	})

	t.Run("missing token info is unauthorized", func(t *testing.T) {

		h := &AuthHandler{}

		_, err := h.ListSessions(t.Context(), api.ListSessionsParams{})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusUnauthorized)
		}
	})
}

func TestAuthHandler_RevokeSession(t *testing.T) {

	t.Run("happy path returns no content", func(t *testing.T) {

		var gotUserID uint
		var gotSessionID string

		h := &AuthHandler{
			sessionService: &mockSessionService{
				revokeSessionFn: func(_ context.Context, userID uint, sessionID string) error {
					gotUserID, gotSessionID = userID, sessionID
					return nil
				},
			},
		}

		ctx := withTokenInfo(t.Context(), &token.AccessTokenInfo{
			UserID: 7,
			Scopes: []string{"app"},
			SID:    "current",
		})

		res, err := h.RevokeSession(ctx, api.RevokeSessionParams{
			SessionId: uuid.MustParse("0195f0b0-0000-7000-8000-000000000002"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, ok := res.(*api.RevokeSessionNoContent); !ok {
			t.Errorf("res = %T, want RevokeSessionNoContent", res)
		}

		want := "0195f0b0-0000-7000-8000-000000000002"
		if gotUserID != 7 || gotSessionID != want {
			t.Errorf("service args = (%d, %q), want (7, %q)", gotUserID, gotSessionID, want)
		}
	})

	t.Run("foreign or unknown session is not found", func(t *testing.T) {

		h := &AuthHandler{
			sessionService: &mockSessionService{
				revokeSessionFn: func(_ context.Context, _ uint, _ string) error {
					return apperrors.ErrNotFound
				},
			},
		}

		ctx := withTokenInfo(t.Context(), &token.AccessTokenInfo{
			UserID: 7,
			Scopes: []string{"app"},
		})

		_, err := h.RevokeSession(ctx, api.RevokeSessionParams{
			SessionId: uuid.MustParse("0195f0b0-0000-7000-8000-000000000003"),
		})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("ping scoped token is forbidden", func(t *testing.T) {

		h := &AuthHandler{}

		ctx := withTokenInfo(t.Context(), &token.AccessTokenInfo{
			UserID: 7,
			Scopes: []string{"ping"},
		})

		_, err := h.RevokeSession(ctx, api.RevokeSessionParams{
			SessionId: uuid.MustParse("0195f0b0-0000-7000-8000-000000000004"),
		})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusForbidden {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusForbidden)
		}
	})
}
