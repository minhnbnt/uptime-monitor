package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{Key: "test-signing-key"},
	}
}

func TestParser_ExpiredToken(t *testing.T) {

	i := do.New()
	config.RegisterConfig(testConfig())(i)
	config.RegisterJwtConfig(i)
	RegisterProvider(i)

	jp := do.MustInvoke[*Provider](i)

	token, err := jp.NewToken("my-app", map[string]any{
		"sub": "1",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}

	_, err = jp.Validate(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestParser_RoundTrip(t *testing.T) {

	i := do.New()
	config.RegisterConfig(testConfig())(i)
	config.RegisterJwtConfig(i)
	RegisterProvider(i)

	jp := do.MustInvoke[*Provider](i)

	t.Run("new token and validate", func(t *testing.T) {
		token, err := jp.NewToken("my-app", map[string]any{
			"sub":   uint(42),
			"email": "test@example.com",
		})
		if err != nil {
			t.Fatalf("NewToken error: %v", err)
		}

		issuer, err := jp.Validate(token)
		if err != nil {
			t.Fatalf("Validate error: %v", err)
		}
		if issuer != "my-app" {
			t.Errorf("issuer = %q, want my-app", issuer)
		}
	})

	t.Run("empty string token", func(t *testing.T) {
		_, err := jp.Validate("")
		if err == nil {
			t.Fatal("expected error for empty token")
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := jp.Validate("not-a-valid.jwt.token")
		if err == nil {
			t.Fatal("expected error for malformed token")
		}
	})
}

func newToken(claims jwt.MapClaims) *Token {
	return &Token{token: &jwt.Token{Claims: claims}}
}

func TestToken_Subject(t *testing.T) {
	t.Run("missing sub", func(t *testing.T) {
		tok := newToken(jwt.MapClaims{"sub": "1"})
		got, err := tok.Subject()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "1" {
			t.Errorf("got %q, want 1", got)
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		tok := newToken(jwt.MapClaims{"sub": 12345})
		_, err := tok.Subject()
		if err == nil {
			t.Fatal("expected error for non-string sub")
		}
	})
}

func TestToken_Issuer(t *testing.T) {
	t.Run("invalid type", func(t *testing.T) {
		tok := newToken(jwt.MapClaims{"iss": 123})
		_, err := tok.Issuer()
		if err == nil {
			t.Fatal("expected error for non-string iss")
		}
	})
}

func TestToken_JTI(t *testing.T) {
	t.Run("missing jti", func(t *testing.T) {
		tok := newToken(jwt.MapClaims{"sub": "1"})
		_, err := tok.JTI()
		if err == nil {
			t.Fatal("expected error for missing jti")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		tok := newToken(jwt.MapClaims{"jti": 999})
		_, err := tok.JTI()
		if err == nil {
			t.Fatal("expected error for non-string jti")
		}
	})
}

func TestToken_Expiry(t *testing.T) {
	t.Run("missing exp", func(t *testing.T) {
		tok := newToken(jwt.MapClaims{"sub": "1"})
		_, err := tok.Expiry()
		if err == nil {
			t.Fatal("expected error for missing exp")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		tok := newToken(jwt.MapClaims{"exp": "not-a-number"})
		_, err := tok.Expiry()
		if err == nil {
			t.Fatal("expected error for non-numeric exp")
		}
	})

	t.Run("float64 exp", func(t *testing.T) {
		tok := newToken(jwt.MapClaims{"exp": float64(1800000000)})
		got, err := tok.Expiry()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Unix() != 1800000000 {
			t.Errorf("got %d, want 1800000000", got.Unix())
		}
	})
}
