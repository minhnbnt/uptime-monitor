package repository

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

var testRedis *redis.Client

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()
		container, client := testcontainers.StartRedis(ctx)
		defer func() { _ = container.Terminate(ctx) }()
		testRedis = client
	}
	os.Exit(m.Run())
}

func newRepository(tb testing.TB) *RedisServerEventRepository {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	return &RedisServerEventRepository{client: testRedis}
}

func TestIntegration_GetStatus_NotFound(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)

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
	testcontainers.CleanRedis(t, testRedis)

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
	testcontainers.CleanRedis(t, testRedis)

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
	testcontainers.CleanRedis(t, testRedis)

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
	testcontainers.CleanRedis(t, testRedis)

	repo := newRepository(t)
	endpointID := uint(10)

	if err := repo.SetStatus(t.Context(), endpointID, domain.StatusOn); err != nil {
		t.Fatalf("SetStatus error: %v", err)
	}

	ttl, err := testRedis.TTL(t.Context(), statusKey(endpointID)).Result()
	if err != nil {
		t.Fatalf("TTL error: %v", err)
	}
	if ttl < 6*24*time.Hour || ttl > 7*24*time.Hour {
		t.Errorf("TTL = %v, want ~7d", ttl)
	}
}
