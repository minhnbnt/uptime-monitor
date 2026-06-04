package infrastructure

import (
	"os"
	"testing"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
)

func TestJwtParser_RoundTrip(t *testing.T) {
	os.Setenv("JWT_KEY", "test-signing-key-for-jwt-parser-test")
	t.Cleanup(func() { os.Unsetenv("JWT_KEY") })

	i := do.New()
	config.RegisterJwtConfig(i)
	RegisterJwtParser(i)

	jp := do.MustInvoke[*JwtParser](i)

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
