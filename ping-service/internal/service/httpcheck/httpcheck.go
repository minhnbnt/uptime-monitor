package httpcheck

import (
	"context"
	"errors"
	"time"

	"github.com/samber/do/v2"

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

func (c *HTTPChecker) Check(ctx context.Context, sv *dto.Server) (ok bool, err error) {

	params := &dto.CheckParams{
		K8sObjectCheckParams: sv.K8sObjectCheckParams,
		HTTPCheckParams:      sv.HTTPCheckParams,
	}

	cachedDomain, ok, pingErr := c.doHTTP(ctx, params, sv.Timeout)
	if ok {
		return true, nil
	}

	if sv.Kind != "Pod" {
		return false, pingErr
	}

	if _, cErr := c.pingWorker.CheckObjectStatus(ctx, &sv.K8sObjectCheckParams); cErr != nil {
		return false, cErr
	}

	if sErr := c.domainResolver.CheckStale(ctx, sv, cachedDomain); sErr != nil {
		return false, sErr
	}

	return false, pingErr
}

// PingOnce runs a single HTTP probe with no pod fallback or stale-domain
// handling; used for one-shot manual pings.
func (c *HTTPChecker) PingOnce(ctx context.Context, params *dto.CheckParams) (ok bool, err error) {
	_, ok, err = c.doHTTP(ctx, params, 0)
	return ok, err
}

// doHTTP resolves the URL, pings it, and validates the response. It returns
// the cached domain (for stale detection) alongside the probe outcome.
func (c *HTTPChecker) doHTTP(ctx context.Context, params *dto.CheckParams, timeout time.Duration) (cachedDomain string, ok bool, pingErr error) {

	url, err := c.domainResolver.ResolveURL(ctx, params)
	if err != nil {
		return "", false, err
	}

	cachedDomain = url.Hostname()
	resp, pingErr := c.pingClient.Ping(
		ctx, timeout,
		params.HTTPCheckParams.Method,
		url.String(),
	)

	if pingErr == nil {
		cErr := c.responseChecker.CheckResponse(params.HTTPCheckParams, *resp)
		if cErr == nil {
			return cachedDomain, true, nil
		}
		pingErr = cErr
	}

	return cachedDomain, false, pingErr
}
