package httpcheck

import (
	"context"
	"testing"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
)

type mockDomainResolver struct {
	resolveDomainNameFn func(ctx context.Context, params *dto.K8sObjectCheckParams) (string, error)
}

func (m *mockDomainResolver) ResolveDomainName(ctx context.Context, params *dto.K8sObjectCheckParams) (string, error) {
	return m.resolveDomainNameFn(ctx, params)
}

type mockDomainStore struct {
	getFn    func(ctx context.Context, key dto.K8sObjectKey) (string, bool, error)
	setFn    func(ctx context.Context, key dto.K8sObjectKey, domain string) error
	deleteFn func(ctx context.Context, key dto.K8sObjectKey) error
	setKeys  []struct {
		key    dto.K8sObjectKey
		domain string
	}
	deleted []dto.K8sObjectKey
}

func (m *mockDomainStore) Get(ctx context.Context, key dto.K8sObjectKey) (string, bool, error) {
	return m.getFn(ctx, key)
}

func (m *mockDomainStore) Set(ctx context.Context, key dto.K8sObjectKey, domain string) error {
	if m.setKeys != nil {
		m.setKeys = append(m.setKeys, struct {
			key    dto.K8sObjectKey
			domain string
		}{key, domain})
	}
	if m.setFn != nil {
		return m.setFn(ctx, key, domain)
	}
	return nil
}

func (m *mockDomainStore) Delete(ctx context.Context, key dto.K8sObjectKey) error {
	if m.deleted != nil {
		m.deleted = append(m.deleted, key)
	}
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		httpParams *dto.HTTPCheckParams
		want       string
	}{
		{
			name:       "path with leading slash",
			host:       "my-api.default.svc.cluster.local",
			httpParams: &dto.HTTPCheckParams{Port: 8080, EndpointPath: "/health"},
			want:       "http://my-api.default.svc.cluster.local:8080/health",
		},
		{
			name:       "path without leading slash",
			host:       "my-api.default.svc.cluster.local",
			httpParams: &dto.HTTPCheckParams{Port: 8080, EndpointPath: "health"},
			want:       "http://my-api.default.svc.cluster.local:8080/health",
		},
		{
			name:       "empty path",
			host:       "my-api.default.svc.cluster.local",
			httpParams: &dto.HTTPCheckParams{Port: 8080, EndpointPath: ""},
			want:       "http://my-api.default.svc.cluster.local:8080",
		},
		{
			name:       "pod ip host",
			host:       "10.0.0.5",
			httpParams: &dto.HTTPCheckParams{Port: 80, EndpointPath: "/health"},
			want:       "http://10.0.0.5:80/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildURL(tt.host, tt.httpParams)
			if got.String() != tt.want {
				t.Errorf("buildURL() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestResolveURLCaching(t *testing.T) {
	makeSvc := func(resolver *mockDomainResolver, store *mockDomainStore) *DomainResolver {
		return NewDomainResolver(resolver, store, nil)
	}

	makeParams := func(kind string) *dto.CheckParams {
		return &dto.CheckParams{
			K8sObjectCheckParams: dto.K8sObjectCheckParams{
				K8sObjectKey: dto.K8sObjectKey{
					Namespace: "default",
					Kind:      kind,
					ObjectID:  "web-app",
				},
			},
			HTTPCheckParams: &dto.HTTPCheckParams{Port: 8080, EndpointPath: "/health"},
		}
	}

	t.Run("pod cache hit skips resolve", func(t *testing.T) {
		resolveCalls := 0
		s := makeSvc(
			&mockDomainResolver{
				resolveDomainNameFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (string, error) {
					resolveCalls++
					return "10.0.0.9", nil
				},
			},
			&mockDomainStore{
				getFn: func(_ context.Context, _ dto.K8sObjectKey) (string, bool, error) {
					return "10.0.0.5", true, nil
				},
			},
		)

		u, err := s.ResolveURL(t.Context(), makeParams("Pod"))
		if err != nil {
			t.Fatalf("ResolveURL: %v", err)
		}
		if u.String() != "http://10.0.0.5:8080/health" {
			t.Errorf("url = %q, want cached host", u.String())
		}
		if resolveCalls != 0 {
			t.Errorf("expected no resolve on cache hit, got %d calls", resolveCalls)
		}
	})

	t.Run("pod cache miss resolves and writes through", func(t *testing.T) {
		store := &mockDomainStore{
			getFn: func(_ context.Context, _ dto.K8sObjectKey) (string, bool, error) {
				return "", false, nil
			},
			setKeys: []struct {
				key    dto.K8sObjectKey
				domain string
			}{},
		}
		s := makeSvc(
			&mockDomainResolver{
				resolveDomainNameFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (string, error) {
					return "10.0.0.7", nil
				},
			},
			store,
		)

		u, err := s.ResolveURL(t.Context(), makeParams("Pod"))
		if err != nil {
			t.Fatalf("ResolveURL: %v", err)
		}
		if u.String() != "http://10.0.0.7:8080/health" {
			t.Errorf("url = %q, want resolved host", u.String())
		}
		want := []struct {
			key    dto.K8sObjectKey
			domain string
		}{{
			key:    dto.K8sObjectKey{Namespace: "default", Kind: "Pod", ObjectID: "web-app"},
			domain: "10.0.0.7",
		}}
		if len(store.setKeys) != 1 || store.setKeys[0] != want[0] {
			t.Errorf("Set = %+v, want %+v", store.setKeys, want)
		}
	})

	t.Run("non-pod kind bypasses cache", func(t *testing.T) {
		resolveCalls := 0
		getCalls := 0
		s := makeSvc(
			&mockDomainResolver{
				resolveDomainNameFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (string, error) {
					resolveCalls++
					return "web-app.default.svc.cluster.local", nil
				},
			},
			&mockDomainStore{
				getFn: func(_ context.Context, _ dto.K8sObjectKey) (string, bool, error) {
					getCalls++
					return "", false, nil
				},
			},
		)

		for _, kind := range []string{"Service", "StatefulSet"} {
			u, err := s.ResolveURL(t.Context(), makeParams(kind))
			if err != nil {
				t.Fatalf("ResolveURL(%s): %v", kind, err)
			}
			if u.String() != "http://web-app.default.svc.cluster.local:8080/health" {
				t.Errorf("url = %q, want computed dns", u.String())
			}
		}
		if resolveCalls != 2 {
			t.Errorf("expected 2 resolves for non-pod kinds, got %d", resolveCalls)
		}
		if getCalls != 0 {
			t.Errorf("expected no cache Get for non-pod kinds, got %d", getCalls)
		}
	})

	t.Run("empty resolved domain is not cached", func(t *testing.T) {
		store := &mockDomainStore{
			getFn: func(_ context.Context, _ dto.K8sObjectKey) (string, bool, error) {
				return "", false, nil
			},
			setKeys: []struct {
				key    dto.K8sObjectKey
				domain string
			}{},
		}
		s := makeSvc(
			&mockDomainResolver{
				resolveDomainNameFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (string, error) {
					return "", nil
				},
			},
			store,
		)

		u, err := s.ResolveURL(t.Context(), makeParams("Pod"))
		if err != nil {
			t.Fatalf("ResolveURL: %v", err)
		}
		if len(store.setKeys) != 0 {
			t.Errorf("expected no Set for empty domain, got %+v", store.setKeys)
		}
		if u == nil {
			t.Error("expected url to be built")
		}
	})

	t.Run("cache get error falls back to resolve", func(t *testing.T) {
		resolveCalls := 0
		s := makeSvc(
			&mockDomainResolver{
				resolveDomainNameFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (string, error) {
					resolveCalls++
					return "10.0.0.6", nil
				},
			},
			&mockDomainStore{
				getFn: func(_ context.Context, _ dto.K8sObjectKey) (string, bool, error) {
					return "", false, context.Canceled
				},
			},
		)

		u, err := s.ResolveURL(t.Context(), makeParams("Pod"))
		if err != nil {
			t.Fatalf("ResolveURL: %v", err)
		}
		if u.String() != "http://10.0.0.6:8080/health" {
			t.Errorf("url = %q, want resolved host after cache error", u.String())
		}
		if resolveCalls != 1 {
			t.Errorf("expected 1 resolve after cache error, got %d", resolveCalls)
		}
	})
}

func TestCheckStale(t *testing.T) {

	makeServer := func() *dto.Server {
		return dto.NewServer(&domain.Server{
			ID:        1,
			Namespace: "default",
			Kind:      "Pod",
			ObjectID:  "web-app",
		})
	}

	t.Run("domain unchanged returns nil and keeps cache", func(t *testing.T) {
		store := &mockDomainStore{deleted: []dto.K8sObjectKey{}}
		d := NewDomainResolver(
			&mockDomainResolver{
				resolveDomainNameFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (string, error) {
					return "10.0.0.5", nil
				},
			},
			store,
			nil,
		)

		err := d.CheckStale(t.Context(), makeServer(), "10.0.0.5")
		if err != nil {
			t.Errorf("CheckStale = %v, want nil", err)
		}
		if len(store.deleted) != 0 {
			t.Errorf("expected no cache delete, got %+v", store.deleted)
		}
	})

	t.Run("domain changed deletes cache and returns ErrStaleDomain", func(t *testing.T) {
		store := &mockDomainStore{deleted: []dto.K8sObjectKey{}}
		d := NewDomainResolver(
			&mockDomainResolver{
				resolveDomainNameFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (string, error) {
					return "10.0.0.2", nil
				},
			},
			store,
			nil,
		)

		err := d.CheckStale(t.Context(), makeServer(), "10.0.0.5")
		if err != ErrStaleDomain {
			t.Errorf("CheckStale = %v, want ErrStaleDomain", err)
		}
		wantKey := dto.K8sObjectKey{Namespace: "default", Kind: "Pod", ObjectID: "web-app"}
		if len(store.deleted) != 1 || store.deleted[0] != wantKey {
			t.Errorf("expected delete %+v, got %+v", wantKey, store.deleted)
		}
	})

	t.Run("resolve error is returned", func(t *testing.T) {
		d := NewDomainResolver(
			&mockDomainResolver{
				resolveDomainNameFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (string, error) {
					return "", context.Canceled
				},
			},
			&mockDomainStore{},
			nil,
		)

		err := d.CheckStale(t.Context(), makeServer(), "10.0.0.5")
		if err != context.Canceled {
			t.Errorf("CheckStale = %v, want context.Canceled", err)
		}
	})
}
