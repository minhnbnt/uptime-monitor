package token

import (
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/domain"
)

func TestTokenGenerator_AccessTokenClaims(t *testing.T) {
	p, tc := setupProviderWithConfig(t)

	g := &tokenGenerator{provider: p, tokenConfig: tc}

	user := &domain.User{Email: "a@b.com", Username: "u"}
	user.ID = 7

	tokenStr, err := g.GenerateAccessToken(user, []string{"api", "ping"}, "session-jti-123")
	if err != nil {
		t.Fatalf("GenerateAccessToken error: %v", err)
	}

	parsed, err := p.ParseWithIssuer(tokenStr, tc.GetAccessTokenIssuer())
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}

	claims, err := parsed.Claims()
	if err != nil {
		t.Fatalf("claims: %v", err)
	}

	if claims["sid"] != "session-jti-123" {
		t.Errorf("sid = %v, want session-jti-123", claims["sid"])
	}
	if claims["scope"] != "api ping" {
		t.Errorf("scope = %v, want %q", claims["scope"], "api ping")
	}
	if claims["sub"] != "7" {
		t.Errorf("sub = %v, want 7", claims["sub"])
	}

	exp, err := parsed.Expiry()
	if err != nil {
		t.Fatalf("expiry: %v", err)
	}
	if exp.Before(time.Now()) {
		t.Error("access token already expired")
	}
}

func TestTokenGenerator_RefreshTokenHasJTI(t *testing.T) {
	p, tc := setupProviderWithConfig(t)

	g := &tokenGenerator{provider: p, tokenConfig: tc}

	refreshUser := &domain.User{}
	refreshUser.ID = 7
	jti := "test-jti-123"
	tokenStr, err := g.GenerateRefreshToken(refreshUser, jti, 0)
	if err != nil {
		t.Fatalf("GenerateRefreshToken error: %v", err)
	}

	parsed, err := p.ParseWithIssuer(tokenStr, tc.GetRefreshTokenIssuer())
	if err != nil {
		t.Fatalf("parse refresh token: %v", err)
	}

	got, err := parsed.JTI()
	if err != nil {
		t.Fatalf("jti: %v", err)
	}
	if got != jti {
		t.Errorf("jti = %q, want %q", got, jti)
	}
}
