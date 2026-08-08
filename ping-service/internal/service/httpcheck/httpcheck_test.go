package httpcheck

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

type mockPingWorker struct {
	checkObjectStatusFn func(ctx context.Context, params *dto.K8sObjectCheckParams) (bool, error)
}

func (m *mockPingWorker) CheckObjectStatus(ctx context.Context, params *dto.K8sObjectCheckParams) (bool, error) {
	return m.checkObjectStatusFn(ctx, params)
}

func newChecker(pingWorker pingWorker, resolver *mockDomainResolver, store *mockDomainStore) *HTTPChecker {
	return NewHTTPChecker(
		pingWorker,
		NewDomainResolver(resolver, store, logger.NewMockLogger()),
		infrastructure.NewPingClient(http.DefaultClient),
		NewResponseChecker(&infrastructure.BodyChecker{}),
	)
}

func newStore() *mockDomainStore {
	return &mockDomainStore{
		getFn: func(_ context.Context, _ dto.K8sObjectKey) (string, bool, error) {
			return "", false, nil
		},
		setKeys: []struct {
			key    dto.K8sObjectKey
			domain string
		}{},
		deleted: []dto.K8sObjectKey{},
	}
}

// httpResolver resolves to the httptest server host so the built URL hits it.
func httpResolver(server *httptest.Server) *mockDomainResolver {
	host := serverHost(server)
	return &mockDomainResolver{
		resolveDomainNameFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (string, error) {
			return host, nil
		},
	}
}

// staleResolver resolves to the server host on the first call (cached domain)
// and to fresh on the re-resolution in CheckStale.
func staleResolver(server *httptest.Server, fresh string) *mockDomainResolver {
	host := serverHost(server)
	calls := 0
	return &mockDomainResolver{
		resolveDomainNameFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (string, error) {
			calls++
			if calls == 1 {
				return host, nil
			}
			return fresh, nil
		},
	}
}

func serverHost(server *httptest.Server) string {
	u, err := url.Parse(server.URL)
	if err != nil {
		panic(err)
	}
	return u.Hostname()
}

func setServerPort(server *httptest.Server, cfg *domain.ServerHTTPConfig) {
	u, err := url.Parse(server.URL)
	if err != nil {
		panic(err)
	}
	port, _ := strconv.Atoi(u.Port())
	cfg.Port = port
}

func TestCheck(t *testing.T) {

	t.Run("service pings HTTP endpoint and checks response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))
		defer server.Close()

		sv := &domain.Server{
			ID:        2,
			Namespace: "default",
			Kind:      "Service",
			ObjectID:  "my-api",
			Timeout:   5 * time.Second,
			HTTPConfig: &domain.ServerHTTPConfig{
				EndpointPath:  "/health",
				ExpectedCode:  200,
				BodyCheckExpr: `status == "ok"`,
				Method:        "GET",
			},
		}
		setServerPort(server, sv.HTTPConfig)

		c := newChecker(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				t.Fatal("pod status check should not run for http-dns server")
				return false, nil
			},
		}, httpResolver(server), newStore())

		ok, err := c.Check(t.Context(), sv)
		if err != nil {
			t.Fatalf("Check: %v", err)
		}
		if !ok {
			t.Error("expected ok=true")
		}
	})

	t.Run("stale pod ip changed invalidates cache", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		sv := &domain.Server{
			ID:       7,
			Namespace: "default",
			Kind:      "Pod",
			ObjectID:  "web-app",
			Timeout:   5 * time.Second,
			HTTPConfig: &domain.ServerHTTPConfig{
				EndpointPath: "/health",
				ExpectedCode: 200,
				Method:       "GET",
			},
		}
		setServerPort(server, sv.HTTPConfig)

		store := newStore()
		c := newChecker(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return true, nil
			},
		}, staleResolver(server, "10.0.0.2"), store)

		ok, err := c.Check(t.Context(), sv)
		if err != ErrStaleDomain {
			t.Errorf("error = %v, want ErrStaleDomain", err)
		}
		if ok {
			t.Error("expected ok=false")
		}
		wantKey := dto.K8sObjectKey{Namespace: "default", Kind: "Pod", ObjectID: "web-app"}
		if len(store.deleted) != 1 || store.deleted[0] != wantKey {
			t.Errorf("expected domainCache.Delete(%+v), got %+v", wantKey, store.deleted)
		}
	})

	t.Run("stale pod ip but pod gone records off", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		sv := &domain.Server{
			ID:       8,
			Namespace: "default",
			Kind:      "Pod",
			ObjectID:  "web-app",
			Timeout:   5 * time.Second,
			HTTPConfig: &domain.ServerHTTPConfig{
				EndpointPath: "/health",
				ExpectedCode: 200,
				Method:       "GET",
			},
		}
		setServerPort(server, sv.HTTPConfig)

		c := newChecker(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return false, errors.New("pod not found")
			},
		}, httpResolver(server), newStore())

		ok, err := c.Check(t.Context(), sv)
		if err == nil {
			t.Fatal("expected error when pod is gone")
		}
		if ok {
			t.Error("expected ok=false")
		}
	})

	t.Run("stale pod ip unchanged records off without cache invalidation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		sv := &domain.Server{
			ID:       9,
			Namespace: "default",
			Kind:      "Pod",
			ObjectID:  "web-app",
			Timeout:   5 * time.Second,
			HTTPConfig: &domain.ServerHTTPConfig{
				EndpointPath: "/health",
				ExpectedCode: 200,
				Method:       "GET",
			},
		}
		setServerPort(server, sv.HTTPConfig)

		store := newStore()
		c := newChecker(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return true, nil
			},
		}, httpResolver(server), store)

		ok, err := c.Check(t.Context(), sv)
		if err == nil {
			t.Fatal("expected error when ip unchanged")
		}
		if ok {
			t.Error("expected ok=false")
		}
		if len(store.deleted) != 0 {
			t.Errorf("expected domain cache not to be invalidated when ip unchanged, got %+v", store.deleted)
		}
	})
}
