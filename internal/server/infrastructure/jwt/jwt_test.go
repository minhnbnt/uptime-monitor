package jwt

import (
	"testing"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
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
