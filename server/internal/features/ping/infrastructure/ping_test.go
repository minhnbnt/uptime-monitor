package infrastructure

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPing_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	code, err := (&PingWorker{httpClient: http.DefaultClient}).Ping(t.Context(), http.MethodGet, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != http.StatusOK {
		t.Errorf("got %d, want %d", code, http.StatusOK)
	}
}

func TestPing_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	code, err := (&PingWorker{httpClient: http.DefaultClient}).Ping(t.Context(), http.MethodGet, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != http.StatusNotFound {
		t.Errorf("got %d, want %d", code, http.StatusNotFound)
	}
}

func TestPing_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	code, err := (&PingWorker{httpClient: http.DefaultClient}).Ping(t.Context(), http.MethodGet, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != http.StatusInternalServerError {
		t.Errorf("got %d, want %d", code, http.StatusInternalServerError)
	}
}

func TestPing_POSTMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	code, err := (&PingWorker{httpClient: http.DefaultClient}).Ping(t.Context(), http.MethodPost, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != http.StatusCreated {
		t.Errorf("got %d, want %d", code, http.StatusCreated)
	}
}

func TestPing_InvalidURL(t *testing.T) {
	_, err := (&PingWorker{httpClient: http.DefaultClient}).Ping(t.Context(), http.MethodGet, "://bad-url")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestPing_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := (&PingWorker{httpClient: http.DefaultClient}).Ping(ctx, http.MethodGet, "http://localhost:1")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
