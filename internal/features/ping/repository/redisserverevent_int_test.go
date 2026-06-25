package repository

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

var testRedis *redis.Client

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()
		container, client := startRedis(ctx)
		defer func() { _ = container.Terminate(ctx) }()
		testRedis = client
	}
	os.Exit(m.Run())
}

func startRedis(ctx context.Context) (testcontainers.Container, *redis.Client) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:8-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForLog("Ready to accept connections tcp").
			WithStartupTimeout(60 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start redis container: %v\n", err)
		os.Exit(1)
	}

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "container host: %v\n", err)
		os.Exit(1)
	}
	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		fmt.Fprintf(os.Stderr, "container port: %v\n", err)
		os.Exit(1)
	}

	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port.Port()),
	})

	return container, client
}

func newRepository(tb testing.TB) *RedisServerEventRepository {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	return &RedisServerEventRepository{client: testRedis}
}

func cleanRedis(tb testing.TB) {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	if err := testRedis.FlushDB(context.Background()).Err(); err != nil {
		tb.Fatalf("flush db: %v", err)
	}
}

func TestIntegration_GetStatus_NotFound(t *testing.T) {
	cleanRedis(t)

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
	cleanRedis(t)

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
	cleanRedis(t)

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
	cleanRedis(t)

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
	cleanRedis(t)

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
