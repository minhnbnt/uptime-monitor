package service

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/k8sclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
)

type domainResolver interface {
	ResolveDomainName(ctx context.Context, params *dto.K8sObjectCheckParams) (string, error)
}

type domainCacheStore interface {
	Get(ctx context.Context, key dto.K8sObjectKey) (string, bool, error)
	Set(ctx context.Context, key dto.K8sObjectKey, domain string) error
}

type URLResolverService struct {
	k8sClient   domainResolver
	domainCache domainCacheStore
}

func RegisterURLResolverService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*URLResolverService, error) {
		return &URLResolverService{
			k8sClient:   do.MustInvoke[*k8sclient.K8sClient](i),
			domainCache: do.MustInvoke[*scheduler.DomainCache](i),
		}, nil
	})
}

func (s *URLResolverService) ResolveURL(ctx context.Context, params *dto.CheckParams) (*url.URL, error) {

	if params == nil || params.HTTPCheckParams == nil {
		return nil, fmt.Errorf("http check params required")
	}

	host, err := s.resolveDomainCached(ctx, &params.K8sObjectCheckParams)
	if err != nil {
		return nil, err
	}

	return buildURL(host, params.HTTPCheckParams), nil
}

// resolveDomainCached caches only Pod kind: Service/StatefulSet domains are
// computed DNS names (no k8s API call), so caching them is pure overhead.
func (s *URLResolverService) resolveDomainCached(ctx context.Context, params *dto.K8sObjectCheckParams) (string, error) {

	if params.Kind != "Pod" {
		return s.k8sClient.ResolveDomainName(ctx, params)
	}

	key := params.K8sObjectKey
	if domain, ok, err := s.domainCache.Get(ctx, key); err == nil && ok {
		return domain, nil
	}

	domain, err := s.k8sClient.ResolveDomainName(ctx, params)
	if err != nil {
		return "", err
	}

	if domain != "" {
		_ = s.domainCache.Set(ctx, key, domain)
	}

	return domain, nil
}

func (s *URLResolverService) ResolveDomain(ctx context.Context, params *dto.K8sObjectCheckParams) (string, error) {
	return s.k8sClient.ResolveDomainName(ctx, params)
}

func buildURL(host string, httpParams *dto.HTTPCheckParams) *url.URL {
	return &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, fmt.Sprint(httpParams.Port)),
		Path:   strings.TrimPrefix(httpParams.EndpointPath, "/"),
	}
}
