package repository

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/jwt"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

var testRedis *redis.Client
var testDB *gorm.DB

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()

		redisContainer, client := testcontainers.StartRedis(ctx)
		defer func() { _ = redisContainer.Terminate(ctx) }()
		testRedis = client

		pgContainer, dsn := testcontainers.StartPostgres(ctx)
		defer func() { _ = pgContainer.Terminate(ctx) }()

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			TranslateError: true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "gorm open: %v\n", err)
			os.Exit(1)
		}

		if err := config.RunMigration(db); err != nil {
			fmt.Fprintf(os.Stderr, "run migration: %v\n", err)
			os.Exit(1)
		}

		testDB = db
	}
	os.Exit(m.Run())
}

func newRevokedRepo(tb testing.TB) *RedisRevokedTokenRepository {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	return &RedisRevokedTokenRepository{client: testRedis}
}

func makeToken(tb testing.TB, jti string, exp time.Time) *jwt.Token {
	tb.Helper()

	i := do.New()
	config.RegisterConfig(&config.Config{
		JWT: config.JWTConfig{Key: "test-key"},
	})(i)
	config.RegisterJwtConfig(i)
	jwt.RegisterProvider(i)
	p := do.MustInvoke[*jwt.Provider](i)

	tokenStr, err := p.NewToken("test", map[string]any{
		"jti": jti,
		"exp": exp.Unix(),
		"sub": "1",
	})
	if err != nil {
		tb.Fatalf("NewToken: %v", err)
	}

	token, err := p.ParseWithIssuer(tokenStr, "test")
	if err != nil {
		tb.Fatalf("ParseWithIssuer: %v", err)
	}
	return token
}

func TestIntegration_RevokeAndIsRevoked(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)

	repo := newRevokedRepo(t)
	jti := "0195f0b0-0000-7000-8000-000000000001"
	token := makeToken(t, jti, time.Now().Add(time.Hour))

	err := repo.Revoke(t.Context(), token)
	if err != nil {
		t.Fatalf("Revoke error: %v", err)
	}

	revoked, err := repo.IsRevoked(t.Context(), jti)
	if err != nil {
		t.Fatalf("IsRevoked error: %v", err)
	}
	if !revoked {
		t.Fatal("expected token to be revoked")
	}
}

func TestIntegration_IsRevoked_NotFound(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)

	repo := newRevokedRepo(t)
	revoked, err := repo.IsRevoked(t.Context(), "nonexistent-jti")
	if err != nil {
		t.Fatalf("IsRevoked error: %v", err)
	}
	if revoked {
		t.Fatal("expected false for non-existent token")
	}
}

func TestIntegration_Revoke_TTL(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)

	repo := newRevokedRepo(t)
	jti := "0195f0b0-0000-7000-8000-000000000002"
	expiry := time.Now().Add(2 * time.Hour)
	token := makeToken(t, jti, expiry)

	err := repo.Revoke(t.Context(), token)
	if err != nil {
		t.Fatalf("Revoke error: %v", err)
	}

	ttl, err := testRedis.TTL(t.Context(), "revoked_token:"+jti).Result()
	if err != nil {
		t.Fatalf("TTL error: %v", err)
	}
	if ttl < time.Hour || ttl > 3*time.Hour {
		t.Errorf("TTL = %v, want ~2h", ttl)
	}
}

func TestIntegration_Revoke_MultipleTokens(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)

	repo := newRevokedRepo(t)

	token1 := makeToken(t, "jti-001", time.Now().Add(time.Hour))
	token2 := makeToken(t, "jti-002", time.Now().Add(time.Hour))

	if err := repo.Revoke(t.Context(), token1); err != nil {
		t.Fatalf("Revoke token1 error: %v", err)
	}
	if err := repo.Revoke(t.Context(), token2); err != nil {
		t.Fatalf("Revoke token2 error: %v", err)
	}

	for _, jti := range []string{"jti-001", "jti-002"} {
		revoked, err := repo.IsRevoked(t.Context(), jti)
		if err != nil {
			t.Fatalf("IsRevoked(%q) error: %v", jti, err)
		}
		if !revoked {
			t.Errorf("expected %q to be revoked", jti)
		}
	}
}
