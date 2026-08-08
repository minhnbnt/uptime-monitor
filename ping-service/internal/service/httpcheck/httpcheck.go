package httpcheck

import (
	"context"
	"errors"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/k8sclient"
)

var ErrStaleDomain = errors.New("stale cached domain, invalidated; skipping event")

type pingWorker interface {
	CheckObjectStatus(ctx context.Context, params *dto.K8sObjectCheckParams) (running bool, err error)
}

type HTTPChecker struct {
	pingWorker      pingWorker
	domainResolver  *DomainResolver
	pingClient      *infrastructure.PingClient
	responseChecker *ResponseChecker
}

func NewHTTPChecker(
	pingWorker pingWorker,
	domainResolver *DomainResolver,
	pingClient *infrastructure.PingClient,
	responseChecker *ResponseChecker,
) *HTTPChecker {
	return &HTTPChecker{
		pingWorker:      pingWorker,
		domainResolver:  domainResolver,
		pingClient:      pingClient,
		responseChecker: responseChecker,
	}
}

func RegisterHTTPChecker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*HTTPChecker, error) {
		return NewHTTPChecker(
			do.MustInvoke[*k8sclient.K8sClient](i),
			do.MustInvoke[*DomainResolver](i),
			do.MustInvoke[*infrastructure.PingClient](i),
			do.MustInvoke[*ResponseChecker](i),
		), nil
	})
}

func (c *HTTPChecker) Check(ctx context.Context, sv *domain.Server) (ok bool, err error) {

	k8sParams := dto.NewK8sObjectCheckParams(sv)

	httpParams := &dto.HTTPCheckParams{
		Method:        sv.HTTPConfig.Method,
		Port:          sv.HTTPConfig.Port,
		EndpointPath:  sv.HTTPConfig.EndpointPath,
		ExpectedCode:  sv.HTTPConfig.ExpectedCode,
		BodyCheckExpr: sv.HTTPConfig.BodyCheckExpr,
	}

	params := &dto.CheckParams{
		K8sObjectCheckParams: *k8sParams,
		HTTPCheckParams:      httpParams,
	}

	url, err := c.domainResolver.ResolveURL(ctx, params)
	if err != nil {
		return false, err
	}
	cachedDomain := url.Hostname()

	resp, pingErr := c.pingClient.Ping(
		ctx, sv.Timeout,
		httpParams.Method,
		url.String(),
	)

	if pingErr == nil {
		cErr := c.responseChecker.CheckResponse(httpParams, *resp)
		if cErr == nil {
			return true, nil
		}
		pingErr = cErr
	}

	if k8sParams.Kind != "Pod" {
		return false, pingErr
	}

	if _, cErr := c.pingWorker.CheckObjectStatus(ctx, k8sParams); cErr != nil {
		return false, cErr
	}

	if sErr := c.domainResolver.CheckStale(ctx, sv, cachedDomain); sErr != nil {
		return false, sErr
	}

	return false, pingErr
}
