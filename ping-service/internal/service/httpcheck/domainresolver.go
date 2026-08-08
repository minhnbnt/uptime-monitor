package httpcheck

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/k8sclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
)

type domainResolver interface {
	ResolveDomainName(ctx context.Context, params *dto.K8sObjectCheckParams) (string, error)
}

type domainCacheStore interface {
	Get(ctx context.Context, key dto.K8sObjectKey) (domain string, ok bool, err error)
	Set(ctx context.Context, key dto.K8sObjectKey, domain string) error
	Delete(ctx context.Context, key dto.K8sObjectKey) error
}

type DomainResolver struct {
	k8sClient   domainResolver
	domainCache domainCacheStore
	logger      *slog.Logger
}

func NewDomainResolver(
	k8sClient domainResolver,
	domainCache domainCacheStore,
	logger *slog.Logger,
) *DomainResolver {
	return &DomainResolver{
		k8sClient:   k8sClient,
		domainCache: domainCache,
		logger:      logger,
	}
}

func RegisterDomainResolver(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*DomainResolver, error) {
		return NewDomainResolver(
			do.MustInvoke[*k8sclient.K8sClient](i),
			do.MustInvoke[*scheduler.DomainCache](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

func (d *DomainResolver) ResolveURL(ctx context.Context, params *dto.CheckParams) (*url.URL, error) {

	if params == nil || params.HTTPCheckParams == nil {
		return nil, fmt.Errorf("http check params required")
	}

	host, err := d.resolveDomainCached(ctx, &params.K8sObjectCheckParams)
	if err != nil {
		return nil, err
	}

	return buildURL(host, params.HTTPCheckParams), nil
}

// resolveDomainCached caches only Pod kind: Service/StatefulSet domains are
// computed DNS names (no k8s API call), so caching them is pure overhead.
func (d *DomainResolver) resolveDomainCached(ctx context.Context, params *dto.K8sObjectCheckParams) (string, error) {

	if params.Kind != "Pod" {
		return d.k8sClient.ResolveDomainName(ctx, params)
	}

	key := params.K8sObjectKey
	if domain, ok, err := d.domainCache.Get(ctx, key); err == nil && ok {
		return domain, nil
	}

	domain, err := d.k8sClient.ResolveDomainName(ctx, params)
	if err != nil {
		return "", err
	}

	if domain != "" {
		_ = d.domainCache.Set(ctx, key, domain)
	}

	return domain, nil
}

// CheckStale re-resolves the domain; if it changed from the cached domain,
// invalidates the cache and returns ErrStaleDomain. Returns nil if unchanged.
func (d *DomainResolver) CheckStale(ctx context.Context, sv *domain.Server, cachedDomain string) error {

	k8sParams := dto.NewK8sObjectCheckParams(sv)

	freshDomain, err := d.k8sClient.ResolveDomainName(ctx, k8sParams)
	if err != nil {
		return err
	}

	if freshDomain == cachedDomain {
		return nil
	}

	if dErr := d.domainCache.Delete(ctx, k8sParams.K8sObjectKey); dErr != nil {
		d.logger.Error(
			"failed to invalidate stale domain cache",
			slog.Uint64("server_id", uint64(sv.ID)),
			slog.Any("error", dErr),
		)
	}

	return ErrStaleDomain
}

func buildURL(host string, httpParams *dto.HTTPCheckParams) *url.URL {
	return &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, fmt.Sprint(httpParams.Port)),
		Path:   strings.TrimPrefix(httpParams.EndpointPath, "/"),
	}
}
