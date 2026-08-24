package authclient

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func authedChain(t *testing.T, headers map[string]string) (*httptest.ResponseRecorder, context.Context) {
	t.Helper()

	var capturedCtx context.Context
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	NewAuthMiddleware(slog.Default()).
		XUserIDMiddleware(next).
		ServeHTTP(rec, req)

	return rec, capturedCtx
}

func guardedChain(t *testing.T, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()

	m := NewAuthMiddleware(slog.Default())
	m.XUserIDMiddleware(m.RequireScope("ping")(next)).ServeHTTP(rec, req)

	return rec
}

func TestXUserIDMiddleware_ParsesScopes(t *testing.T) {
	rec, ctx := authedChain(t, map[string]string{
		"X-User-ID": "42",
		"X-Scopes":  "app ping",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if GetUserID(ctx) != 42 {
		t.Errorf("userID = %d, want 42", GetUserID(ctx))
	}
	if !HasScope(ctx, "app") || !HasScope(ctx, "ping") {
		t.Errorf("scopes = %v, want [app ping]", GetScopes(ctx))
	}
}

func TestXUserIDMiddleware_NoScopesHeader(t *testing.T) {
	rec, ctx := authedChain(t, map[string]string{"X-User-ID": "7"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if GetUserID(ctx) != 7 {
		t.Errorf("userID = %d, want 7", GetUserID(ctx))
	}
	if HasScope(ctx, "app") {
		t.Error("expected no scopes")
	}
}

func TestRequireScope_AllowedWithScope(t *testing.T) {
	rec := guardedChain(t, map[string]string{
		"X-User-ID": "42",
		"X-Scopes":  "ping",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestRequireScope_ForbiddenWithoutScope(t *testing.T) {
	rec := guardedChain(t, map[string]string{"X-User-ID": "7"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestRequireScope_ForbiddenUnauthenticated(t *testing.T) {
	rec := guardedChain(t, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}
