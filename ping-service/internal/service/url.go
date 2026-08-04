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
)

type URLResolverService struct {
	k8sClient *k8sclient.K8sClient
}

func RegisterURLResolverService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*URLResolverService, error) {
		return &URLResolverService{
			k8sClient: do.MustInvoke[*k8sclient.K8sClient](i),
		}, nil
	})
}

func (s *URLResolverService) ResolveURL(ctx context.Context, params *dto.CheckParams) (*url.URL, error) {

	if params == nil || params.HTTPCheckParams == nil {
		return nil, fmt.Errorf("http check params required")
	}

	host, err := s.k8sClient.ResolveDomainName(ctx, &params.K8sObjectCheckParams)
	if err != nil {
		return nil, err
	}

	return buildURL(host, params.HTTPCheckParams), nil
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
