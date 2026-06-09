package auth

import (
	"os"
	"testing"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
)

func setupTokenValidatorTest(t *testing.T) *TokenValidator {
	t.Helper()

	os.Setenv("JWT_KEY", "test-signing-key-for-token-validator")
	t.Cleanup(func() { os.Unsetenv("JWT_KEY") })

	i := do.New()
	config.RegisterJwtConfig(i)
	config.RegisterTokenConfig(i)
	jwtutil.RegisterProvider(i)
	RegisterTokenValidator(i)

	return do.MustInvoke[*TokenValidator](i)
}

func setupProviderWithConfig(t *testing.T) (*jwtutil.Provider, *config.TokenConfig) {
	t.Helper()

	os.Setenv("JWT_KEY", "test-signing-key-for-token-validator")
	t.Cleanup(func() { os.Unsetenv("JWT_KEY") })

	i := do.New()
	config.RegisterJwtConfig(i)
	config.RegisterTokenConfig(i)
	jwtutil.RegisterProvider(i)

	return do.MustInvoke[*jwtutil.Provider](i), do.MustInvoke[*config.TokenConfig](i)
}

func TestValidateAccessToken_Success(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc}

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
	tv := &TokenValidator{provider: p, tokenConfig: tc}

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
	tv := &TokenValidator{provider: p, tokenConfig: tc}

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
	tv := &TokenValidator{provider: p, tokenConfig: tc}

	_, err := tv.ValidateAccessToken("not-a-valid-token")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestValidateRefreshToken_Success(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc}

	token, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": "42",
		"jti": "0195f0b0-0000-7000-8000-000000000000",
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	userID, jti, err := tv.ValidateRefreshToken(token)
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
	tv := &TokenValidator{provider: p, tokenConfig: tc}

	token, err := p.NewToken(tc.GetAccessTokenIssuer(), map[string]any{
		"sub": "42",
		"jti": "0195f0b0-0000-7000-8000-000000000000",
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, _, err = tv.ValidateRefreshToken(token)
	if err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestValidateRefreshToken_Expired(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc}

	token, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": "42",
		"jti": "0195f0b0-0000-7000-8000-000000000000",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, _, err = tv.ValidateRefreshToken(token)
	if err == nil {
		t.Fatal("expected error for expired refresh token")
	}
}

func TestValidateRefreshToken_MissingJTI(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc}

	token, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": "42",
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, _, err = tv.ValidateRefreshToken(token)
	if err == nil {
		t.Fatal("expected error for missing jti")
	}
}

func TestValidateRefreshToken_Malformed(t *testing.T) {
	p, tc := setupProviderWithConfig(t)
	tv := &TokenValidator{provider: p, tokenConfig: tc}

	_, _, err := tv.ValidateRefreshToken("not-a-valid-token")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}
