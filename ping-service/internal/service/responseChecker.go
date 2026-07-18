package service

import (
	"fmt"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	infra "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
)

type ResponseChecker struct {
	bodyChecker *infra.BodyChecker
}

func RegisterResponseChecker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ResponseChecker, error) {
		return &ResponseChecker{bodyChecker: do.MustInvoke[*infra.BodyChecker](i)}, nil
	})
}

func (rc *ResponseChecker) CheckResponse(endpoint domain.Endpoint, response infra.Response) error {

	if endpoint.ExpectedCode != response.StatusCode {
		return fmt.Errorf(
			"unexpected status code: got %d, want %d",
			response.StatusCode, endpoint.ExpectedCode,
		)
	}

	if endpoint.BodyCheckExpr == nil || *endpoint.BodyCheckExpr == "" {
		return nil
	}

	ok, err := rc.bodyChecker.Check(response.Body, *endpoint.BodyCheckExpr)
	if err != nil {
		return fmt.Errorf("body check failed: %w", err)
	}
	if !ok {
		return fmt.Errorf(
			"body check expression evaluated to false: %q",
			*endpoint.BodyCheckExpr,
		)
	}

	return nil
}
