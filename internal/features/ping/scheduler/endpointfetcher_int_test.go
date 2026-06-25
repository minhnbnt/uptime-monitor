package scheduler

import (
	"testing"
)

func newFetcher(tb testing.TB) *EndpointFetcher {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	return &EndpointFetcher{db: testDB}
}

func TestIntegration_Fetch_EmptyIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	truncateTables(t)

	f := newFetcher(t)
	results, err := f.Fetch(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("results = %v, want nil", results)
	}
}

func TestIntegration_Fetch_SingleID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	truncateTables(t)
	seedServer(t, 1)
	seedEndpoint(t, 100, 1)

	f := newFetcher(t)
	results, err := f.Fetch(t.Context(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].ID != 100 {
		t.Errorf("ID = %d, want 100", results[0].ID)
	}
	if results[0].URL != "https://example-100.com" {
		t.Errorf("URL = %q", results[0].URL)
	}
}

func TestIntegration_Fetch_MultipleIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	truncateTables(t)
	seedServer(t, 1)
	seedEndpoint(t, 1, 1)
	seedEndpoint(t, 2, 1)
	seedEndpoint(t, 3, 1)

	f := newFetcher(t)
	results, err := f.Fetch(t.Context(), 1, 2, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
}

func TestIntegration_Fetch_NonExistentID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	truncateTables(t)
	seedServer(t, 1)
	seedEndpoint(t, 1, 1)

	f := newFetcher(t)
	results, err := f.Fetch(t.Context(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}

func TestIntegration_Fetch_MixedExistingAndMissing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	truncateTables(t)
	seedServer(t, 1)
	seedEndpoint(t, 1, 1)
	seedEndpoint(t, 2, 1)

	f := newFetcher(t)
	results, err := f.Fetch(t.Context(), 1, 999, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
}
