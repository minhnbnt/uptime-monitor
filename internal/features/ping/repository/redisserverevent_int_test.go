package repository

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

var testRedis *redis.Client
var testRedisAddr string

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()
		container, addr := testcontainers.StartRedisAddr(ctx)
		defer func() { _ = container.Terminate(ctx) }()
		testRedisAddr = addr
	}
	os.Exit(m.Run())
}

func newRepository(tb testing.TB) *RedisServerEventRepository {
	tb.Helper()
	testcontainers.SkipIfShort(tb)
	return &RedisServerEventRepository{client: testRedis}
}

func TestIntegration_GetStatus_NotFound(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testRedis = testcontainers.NewTestRedis(t, testRedisAddr)

	repo := newRepository(t)
	got, err := repo.GetStatus(t.Context(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestIntegration_SetGetStatus(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testRedis = testcontainers.NewTestRedis(t, testRedisAddr)

	repo := newRepository(t)
	endpointID := uint(42)

	err := repo.SetStatus(t.Context(), endpointID, domain.StatusOn)
	if err != nil {
		t.Fatalf("SetStatus error: %v", err)
	}

	got, err := repo.GetStatus(t.Context(), endpointID)
	if err != nil {
		t.Fatalf("GetStatus error: %v", err)
	}
	if got != domain.StatusOn {
		t.Errorf("got %q, want %q", got, domain.StatusOn)
	}
}

func TestIntegration_SetStatus_Overwrite(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testRedis = testcontainers.NewTestRedis(t, testRedisAddr)

	repo := newRepository(t)
	endpointID := uint(1)

	if err := repo.SetStatus(t.Context(), endpointID, domain.StatusOn); err != nil {
		t.Fatalf("SetStatus(On) error: %v", err)
	}
	if err := repo.SetStatus(t.Context(), endpointID, domain.StatusOff); err != nil {
		t.Fatalf("SetStatus(Off) error: %v", err)
	}

	got, err := repo.GetStatus(t.Context(), endpointID)
	if err != nil {
		t.Fatalf("GetStatus error: %v", err)
	}
	if got != domain.StatusOff {
		t.Errorf("got %q, want %q", got, domain.StatusOff)
	}
}

func TestIntegration_DeleteStatus(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testRedis = testcontainers.NewTestRedis(t, testRedisAddr)

	repo := newRepository(t)
	endpointID := uint(7)

	if err := repo.SetStatus(t.Context(), endpointID, domain.StatusOn); err != nil {
		t.Fatalf("SetStatus error: %v", err)
	}
	if err := repo.DeleteStatus(t.Context(), endpointID); err != nil {
		t.Fatalf("DeleteStatus error: %v", err)
	}

	got, err := repo.GetStatus(t.Context(), endpointID)
	if err != nil {
		t.Fatalf("GetStatus error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty (deleted)", got)
	}
}

func TestIntegration_SetStatus_TTL(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testRedis = testcontainers.NewTestRedis(t, testRedisAddr)

	repo := newRepository(t)
	endpointID := uint(10)

	if err := repo.SetStatus(t.Context(), endpointID, domain.StatusOn); err != nil {
		t.Fatalf("SetStatus error: %v", err)
	}

	ttls, err := testRedis.HExpireTime(t.Context(), statusKey, fmt.Sprint(endpointID)).Result()
	if err != nil {
		t.Fatalf("HExpireTime error: %v", err)
	}
	if len(ttls) != 1 {
		t.Fatalf("expected 1 TTL result, got %d", len(ttls))
	}
	if ttls[0] < 0 {
		t.Fatalf("field has no TTL set")
	}
	got := time.Until(time.Unix(ttls[0], 0))
	if got < 6*24*time.Hour || got > 7*24*time.Hour+time.Minute {
		t.Errorf("TTL = %v, want ~7d", got)
	}
}
