package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"

	pingv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/ping/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/k8sclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/service"
)

type PingServer struct {
	pingv1.UnimplementedPingServiceServer
	k8sClient       *k8sclient.K8sClient
	urlResolver     *service.URLResolverService
	pingClient      *infrastructure.PingClient
	responseChecker *service.ResponseChecker
}

func RegisterPingServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingServer, error) {
		return &PingServer{
			k8sClient:       do.MustInvoke[*k8sclient.K8sClient](i),
			urlResolver:     do.MustInvoke[*service.URLResolverService](i),
			pingClient:      do.MustInvoke[*infrastructure.PingClient](i),
			responseChecker: do.MustInvoke[*service.ResponseChecker](i),
		}, nil
	})
}

func (s *PingServer) Ping(ctx context.Context, req *pingv1.PingRequest) (*pingv1.PingResponse, error) {

	timeout := time.Duration(req.TimeoutMs) * time.Millisecond
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	k8sParams := &dto.K8sObjectCheckParams{
		Namespace:     req.Namespace,
		Kind:          req.Kind,
		ObjectID:      req.ObjectId,
		ContainerName: req.ContainerName,
	}

	running, err := s.doPing(ctx, k8sParams, req)
	if err != nil {
		return &pingv1.PingResponse{
			Running: false,
			Error:   fmt.Sprintf("check error: %s", err.Error()),
		}, nil
	}

	return &pingv1.PingResponse{Running: running}, nil
}

func (s *PingServer) doPing(ctx context.Context, k8sParams *dto.K8sObjectCheckParams, req *pingv1.PingRequest) (bool, error) {

	switch req.PingType {

	case pingv1.PingType_PING_TYPE_HTTP_DNS:
		return s.checkHTTPDNS(ctx, k8sParams, req.GetHttpDnsConfig())

	default:
		return s.k8sClient.CheckObjectStatus(ctx, k8sParams)
	}
}

func (s *PingServer) checkHTTPDNS(ctx context.Context, k8sParams *dto.K8sObjectCheckParams, cfg *pingv1.HttpDnsConfig) (bool, error) {

	if cfg == nil {
		return false, fmt.Errorf("http-dns config required")
	}

	httpParams := &dto.HTTPCheckParams{
		Method:        cfg.Method,
		Port:          int(cfg.Port),
		EndpointPath:  cfg.EndpointPath,
		ExpectedCode:  int(cfg.ExpectedCode),
		BodyCheckExpr: cfg.BodyCheckExpr,
	}

	params := &dto.CheckParams{
		K8sObjectCheckParams: *k8sParams,
		HTTPCheckParams:      httpParams,
	}

	url, err := s.urlResolver.ResolveURL(ctx, params)
	if err != nil {
		return false, err
	}

	resp, err := s.pingClient.Ping(ctx, 0, httpParams.Method, url.String())
	if err != nil {
		return false, err
	}

	if err := s.responseChecker.CheckResponse(httpParams, *resp); err != nil {
		return false, err
	}

	return true, nil
}
