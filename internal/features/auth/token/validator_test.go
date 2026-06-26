package token

import (
	"context"
	"testing"
	"time"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/jwt"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{Key: "test-signing-key-for-token-validator"},
		Token: config.TokenCfg{
			AccessTTL:     "15m",
			RefreshTTL:    "168h",
			AccessIssuer:  "uptime-monitor",
			RefreshIssuer: "uptime-monitor-refresh",
		},
	}
}

func setupProviderWithConfig(t *testing.T) (*jwt.Provider, *config.TokenConfig) {
	t.Helper()

	i := do.New()
	config.RegisterConfig(testConfig())(i)
	config.RegisterJwtConfig(i)
	config.RegisterTokenConfig(i)
	jwt.RegisterProvider(i)

	return do.MustInvoke[*jwt.Provider](i), do.MustInvoke[*config.TokenConfig](i)
}

func TestValidateAccessToken_Success(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, logger: logger.NewMockLogger()}

	token, err := p.NewToken(tc.GetAccessTokenIssuer(), map[string]any{
		"sub":      "42",
		"email":    "test@example.com",
		"username": "testuser",
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	userID, err := tv.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken error: %v", err)
	}
	if userID != 42 {
		t.Errorf("userID = %d, want 42", userID)
	}
}

func TestValidateAccessToken_WrongIssuer(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, logger: logger.NewMockLogger()}

	token, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": "42",
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, err = tv.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, logger: logger.NewMockLogger()}

	token, err := p.NewToken(tc.GetAccessTokenIssuer(), map[string]any{
		"sub": "42",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, err = tv.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateAccessToken_Malformed(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, logger: logger.NewMockLogger()}

	_, err := tv.ValidateAccessToken("not-a-valid-token")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestValidateRefreshToken_Success(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, revokedTokenRepo: &mockRevokedTokenRepo{}, logger: logger.NewMockLogger()}

	token, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": "42",
		"jti": "0195f0b0-0000-7000-8000-000000000000",
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	userID, jti, err := tv.ValidateRefreshToken(t.Context(), token)
	if err != nil {
		t.Fatalf("ValidateRefreshToken error: %v", err)
	}
	if userID != 42 {
		t.Errorf("userID = %d, want 42", userID)
	}
	if jti != "0195f0b0-0000-7000-8000-000000000000" {
		t.Errorf("jti = %q, want 0195f0b0-0000-7000-8000-000000000000", jti)
	}
}

func TestValidateRefreshToken_WrongIssuer(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, revokedTokenRepo: &mockRevokedTokenRepo{}, logger: logger.NewMockLogger()}

	token, err := p.NewToken(tc.GetAccessTokenIssuer(), map[string]any{
		"sub": "42",
		"jti": "0195f0b0-0000-7000-8000-000000000000",
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, _, err = tv.ValidateRefreshToken(t.Context(), token)
	if err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestValidateRefreshToken_Expired(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, revokedTokenRepo: &mockRevokedTokenRepo{}, logger: logger.NewMockLogger()}

	token, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": "42",
		"jti": "0195f0b0-0000-7000-8000-000000000000",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, _, err = tv.ValidateRefreshToken(t.Context(), token)
	if err == nil {
		t.Fatal("expected error for expired refresh token")
	}
}

func TestValidateRefreshToken_MissingJTI(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, revokedTokenRepo: &mockRevokedTokenRepo{}, logger: logger.NewMockLogger()}

	token, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": "42",
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, _, err = tv.ValidateRefreshToken(t.Context(), token)
	if err == nil {
		t.Fatal("expected error for missing jti")
	}
}

func TestValidateRefreshToken_Malformed(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, revokedTokenRepo: &mockRevokedTokenRepo{}, logger: logger.NewMockLogger()}

	_, _, err := tv.ValidateRefreshToken(t.Context(), "not-a-valid-token")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestValidateAccessToken_InvalidSubject(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, logger: logger.NewMockLogger()}

	token, err := p.NewToken(tc.GetAccessTokenIssuer(), map[string]any{
		"sub": 12345,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, err = tv.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("expected error for invalid subject")
	}
}

func TestValidateAccessToken_NonNumericSubject(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, logger: logger.NewMockLogger()}

	token, err := p.NewToken(tc.GetAccessTokenIssuer(), map[string]any{
		"sub": "not-a-number",
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, err = tv.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("expected error for non-numeric subject")
	}
}

func TestValidateRefreshToken_InvalidSubject(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, revokedTokenRepo: &mockRevokedTokenRepo{}, logger: logger.NewMockLogger()}

	token, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": 12345,
		"jti": "0195f0b0-0000-7000-8000-000000000000",
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, _, err = tv.ValidateRefreshToken(t.Context(), token)
	if err == nil {
		t.Fatal("expected error for invalid subject")
	}
}

func TestValidateRefreshToken_NonNumericSubject(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, revokedTokenRepo: &mockRevokedTokenRepo{}, logger: logger.NewMockLogger()}

	token, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": "not-a-number",
		"jti": "0195f0b0-0000-7000-8000-000000000000",
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, _, err = tv.ValidateRefreshToken(t.Context(), token)
	if err == nil {
		t.Fatal("expected error for non-numeric subject")
	}
}

func TestValidateRefreshToken_IsRevokedError(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{
		provider:    p,
		tokenConfig: tc,
		revokedTokenRepo: &mockRevokedTokenRepo{
			isRevokedFn: func(_ context.Context, _ string) (bool, error) {
				return false, assert.AnError
			},
		},
		logger: logger.NewMockLogger(),
	}

	token, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": "42",
		"jti": "0195f0b0-0000-7000-8000-000000000000",
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, _, err = tv.ValidateRefreshToken(t.Context(), token)
	if err == nil {
		t.Fatal("expected error when IsRevoked fails")
	}
}

func TestParseRefreshToken_Success(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, logger: logger.NewMockLogger()}

	tokenStr, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": "42",
		"jti": "0195f0b0-0000-7000-8000-000000000000",
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	token, err := tv.ParseRefreshToken(tokenStr)
	if err != nil {
		t.Fatalf("ParseRefreshToken error: %v", err)
	}
	sub, _ := token.Subject()
	if sub != "42" {
		t.Errorf("subject = %q, want 42", sub)
	}
}

func TestParseRefreshToken_WrongIssuer(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, logger: logger.NewMockLogger()}

	tokenStr, err := p.NewToken(tc.GetAccessTokenIssuer(), map[string]any{
		"sub": "42",
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, err = tv.ParseRefreshToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestParseRefreshToken_Expired(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, logger: logger.NewMockLogger()}

	tokenStr, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": "42",
		"jti": "0195f0b0-0000-7000-8000-000000000000",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, err = tv.ParseRefreshToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestParseRefreshToken_Malformed(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc, logger: logger.NewMockLogger()}

	_, err := tv.ParseRefreshToken("not-a-valid-token")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestValidateRefreshToken_Revoked(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{
		provider:    p,
		tokenConfig: tc,
		revokedTokenRepo: &mockRevokedTokenRepo{
			isRevokedFn: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
		},
		logger: logger.NewMockLogger(),
	}

	token, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": "42",
		"jti": "0195f0b0-0000-7000-8000-000000000000",
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, _, err = tv.ValidateRefreshToken(t.Context(), token)
	if err == nil {
		t.Fatal("expected error for revoked token")
	}
}
