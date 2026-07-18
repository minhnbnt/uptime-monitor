package grpcserver

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"

	pingv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/ping/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	pinginfra "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/service"
)

type PingServer struct {
	pingv1.UnimplementedPingServiceServer
	pingWorker      *pinginfra.PingWorker
	responseChecker *service.ResponseChecker
}

func RegisterPingServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingServer, error) {
		return &PingServer{
			pingWorker:      do.MustInvoke[*pinginfra.PingWorker](i),
			responseChecker: do.MustInvoke[*service.ResponseChecker](i),
		}, nil
	})
}

func (s *PingServer) Ping(ctx context.Context, req *pingv1.PingRequest) (*pingv1.PingResponse, error) {

	timeout := time.Duration(req.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	bodyExpr := req.BodyCheckExpr
	var bodyExprPtr *string
	if bodyExpr != "" {
		bodyExprPtr = &bodyExpr
	}

	endpoint := &domain.Endpoint{
		URL:          req.Url,
		Method:       req.Method,
		Timeout:      timeout,
		ExpectedCode: int(req.ExpectedCode),
		BodyCheckExpr: bodyExprPtr,
	}

	resp, err := s.pingWorker.Ping(ctx, endpoint)
	if err != nil {
		return &pingv1.PingResponse{
			StatusCode: 0,
			Error:      fmt.Sprintf("ping error: %s", err.Error()),
		}, nil
	}

	if err := s.responseChecker.CheckResponse(*endpoint, *resp); err != nil {
		return &pingv1.PingResponse{
			StatusCode: int32(resp.StatusCode),
			Error:      fmt.Sprintf("check failed: %s", err.Error()),
		}, nil
	}

	return &pingv1.PingResponse{
		StatusCode: int32(resp.StatusCode),
	}, nil
}
