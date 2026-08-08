package cache

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/testcontainers"
)

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

func TestServerMetaCacheRoundTrip(t *testing.T) {
	testcontainers.SkipIfShort(t)
	client := testcontainers.NewTestRedis(t, testRedisAddr)
	cache := NewServerMetaCache(client)
	ctx := context.Background()

	sv := &domain.Server{
		ID:            42,
		Namespace:     "default",
		Kind:          "Deployment",
		ObjectID:      "web",
		ContainerName: "app",
		Interval:      30 * time.Second,
		Timeout:       5 * time.Second,
		HTTPConfig: &domain.ServerHTTPConfig{
			Port:         8080,
			EndpointPath: "/health",
			ExpectedCode: 200,
			Method:       "GET",
		},
	}

	if err := cache.Set(ctx, sv); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := cache.Get(ctx, sv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.K8s != nil {
		t.Errorf("expected no K8s runtime in meta cache, got %+v", got.K8s)
	}
	if got.Kind != sv.Kind || got.ObjectID != sv.ObjectID || got.ContainerName != sv.ContainerName {
		t.Errorf("identity mismatch: got %+v", got)
	}
	if got.HTTPConfig == nil || got.HTTPConfig.Port != 8080 || got.HTTPConfig.EndpointPath != "/health" {
		t.Errorf("HTTPConfig mismatch: got %+v", got.HTTPConfig)
	}
}

func TestDomainCacheRoundTrip(t *testing.T) {
	testcontainers.SkipIfShort(t)
	client := testcontainers.NewTestRedis(t, testRedisAddr)
	cache := NewDomainCache(client)
	ctx := context.Background()

	key := dto.K8sObjectKey{Namespace: "default", Kind: "Pod", ObjectID: "web-app"}
	domain := "10.0.0.5"

	if got, ok, err := cache.Get(ctx, key); err != nil || ok {
		t.Errorf("expected miss on empty cache, got %q ok=%v err=%v", got, ok, err)
	}

	if err := cache.Set(ctx, key, ""); err != nil {
		t.Fatalf("Set empty: %v", err)
	}
	if got, ok, err := cache.Get(ctx, key); err != nil || ok {
		t.Errorf("empty Set should not write, got %q ok=%v err=%v", got, ok, err)
	}

	if err := cache.Set(ctx, key, domain); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok, err := cache.Get(ctx, key)
	if err != nil || !ok || got != domain {
		t.Errorf("Get = %q ok=%v err=%v, want %q", got, ok, err, domain)
	}

	if err := cache.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, ok, err := cache.Get(ctx, key); err != nil || ok {
		t.Errorf("expected miss after delete, got %q ok=%v err=%v", got, ok, err)
	}
}
