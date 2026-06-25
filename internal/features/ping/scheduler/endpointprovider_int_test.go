package scheduler

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

func seedCacheEndpoint(tb testing.TB, ep domain.Endpoint) {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	key := metaCacheKey(ep.ID)
	data := map[string]string{
		"url":           ep.URL,
		"method":        ep.Method,
		"expected_code": fmt.Sprint(ep.ExpectedCode),
		"interval_ns":   fmt.Sprint(ep.Interval.Nanoseconds()),
	}
	if err := testRedis.HSet(tb.Context(), key, data).Err(); err != nil {
		tb.Fatalf("seed cache: %v", err)
	}
}

func verifyCacheContains(tb testing.TB, ep domain.Endpoint) {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	key := metaCacheKey(ep.ID)
	data, err := testRedis.HGetAll(tb.Context(), key).Result()
	if err != nil {
		tb.Fatalf("HGetAll: %v", err)
	}
	if len(data) == 0 {
		tb.Errorf("cache key %s is empty", key)
		return
	}
	if data["url"] != ep.URL {
		tb.Errorf("cached url = %q, want %q", data["url"], ep.URL)
	}
	if data["method"] != ep.Method {
		tb.Errorf("cached method = %q, want %q", data["method"], ep.Method)
	}
	intervalNs, _ := strconv.ParseInt(data["interval_ns"], 10, 64)
	if time.Duration(intervalNs) != ep.Interval {
		tb.Errorf("cached interval = %d, want %d", intervalNs, ep.Interval.Nanoseconds())
	}
}

func newEndpointProvider(tb testing.TB) *EndpointProvider {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	return &EndpointProvider{
		cache:   &EndpointMetaCache{client: testRedis},
		fetcher: &EndpointFetcher{db: testDB},
	}
}

func TestIntegration_GetBatch_AllCached(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cleanRedis(t)

	seedCacheEndpoint(t, domain.Endpoint{
		Model:        gorm.Model{ID: 1},
		URL:          "https://cached.com",
		Method:       "GET",
		ExpectedCode: 200,
		Interval:     30 * time.Second,
	})

	p := newEndpointProvider(t)
	results, err := p.GetBatch(t.Context(), []uint{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	ep, ok := results[1]
	if !ok {
		t.Fatal("endpoint 1 not in results")
	}
	if ep.URL != "https://cached.com" {
		t.Errorf("URL = %q, want https://cached.com", ep.URL)
	}
}

func TestIntegration_GetBatch_AllMissed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cleanRedis(t)
	truncateTables(t)

	seedServer(t, 1)
	seedEndpoint(t, 10, 1)

	p := newEndpointProvider(t)
	results, err := p.GetBatch(t.Context(), []uint{10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	ep, ok := results[10]
	if !ok {
		t.Fatal("endpoint 10 not in results")
	}
	if ep.URL != "https://example-10.com" {
		t.Errorf("URL = %q, want https://example-10.com", ep.URL)
	}

	verifyCacheContains(t, domain.Endpoint{
		Model:        gorm.Model{ID: 10},
		URL:          "https://example-10.com",
		Method:       "GET",
		ExpectedCode: 200,
		Interval:     30 * time.Second,
	})
}

func TestIntegration_GetBatch_PartialMiss(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cleanRedis(t)
	truncateTables(t)

	seedCacheEndpoint(t, domain.Endpoint{
		Model:        gorm.Model{ID: 1},
		URL:          "https://cached.com",
		Method:       "GET",
		ExpectedCode: 200,
		Interval:     30 * time.Second,
	})

	seedServer(t, 1)
	seedEndpoint(t, 2, 1)

	p := newEndpointProvider(t)
	results, err := p.GetBatch(t.Context(), []uint{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	ep1, ok := results[1]
	if !ok {
		t.Fatal("endpoint 1 not in results (should be from cache)")
	}
	if ep1.URL != "https://cached.com" {
		t.Errorf("ep1.URL = %q, want https://cached.com", ep1.URL)
	}

	ep2, ok := results[2]
	if !ok {
		t.Fatal("endpoint 2 not in results (should be from db)")
	}
	if ep2.URL != "https://example-2.com" {
		t.Errorf("ep2.URL = %q, want https://example-2.com", ep2.URL)
	}

	verifyCacheContains(t, domain.Endpoint{
		Model:        gorm.Model{ID: 2},
		URL:          "https://example-2.com",
		Method:       "GET",
		ExpectedCode: 200,
		Interval:     30 * time.Second,
	})
}

func TestIntegration_GetBatch_EmptyIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cleanRedis(t)

	p := newEndpointProvider(t)
	results, err := p.GetBatch(t.Context(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}

func TestIntegration_GetBatch_AllMissedMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cleanRedis(t)
	truncateTables(t)

	seedServer(t, 1)
	seedEndpoint(t, 100, 1)
	seedEndpoint(t, 101, 1)

	p := newEndpointProvider(t)
	results, err := p.GetBatch(t.Context(), []uint{100, 101})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	verifyCacheContains(t, domain.Endpoint{
		Model:        gorm.Model{ID: 100},
		URL:          "https://example-100.com",
		Method:       "GET",
		ExpectedCode: 200,
		Interval:     30 * time.Second,
	})
	verifyCacheContains(t, domain.Endpoint{
		Model:        gorm.Model{ID: 101},
		URL:          "https://example-101.com",
		Method:       "GET",
		ExpectedCode: 200,
		Interval:     30 * time.Second,
	})
}
