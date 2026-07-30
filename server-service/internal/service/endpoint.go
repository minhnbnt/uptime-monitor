package service

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"

	pingv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/ping/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/grpcclient"
)

type EndpointService struct {
	pingClient *grpcclient.PingClient
	logger     *slog.Logger
}

func RegisterEndpointService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointService, error) {
		return &EndpointService{
			pingClient: do.MustInvoke[*grpcclient.PingClient](i),
			logger:     do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (es *EndpointService) TestEndpoint(ctx context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error) {

	timeoutMs := int64(req.Timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = 10000
	}

	pingReq := &pingv1.PingRequest{
		Namespace:     req.Namespace,
		Kind:          req.Kind,
		ObjectId:      req.ObjectID,
		ContainerName: req.ContainerName,
		TimeoutMs:     timeoutMs,
	}

	if req.HttpConfig != nil {
		pingReq.PingType = pingv1.PingType_PING_TYPE_HTTP_DNS
		pingReq.HttpDnsConfig = &pingv1.HttpDnsConfig{
			Port:          int32(req.HttpConfig.Port),
			EndpointPath:  req.HttpConfig.EndpointPath,
			ExpectedCode:  int32(req.HttpConfig.ExpectedCode),
			BodyCheckExpr: req.HttpConfig.BodyCheckExpr,
		}
	} else {
		pingReq.PingType = pingv1.PingType_PING_TYPE_POD_STATUS
	}

	running, err := es.pingClient.Ping(ctx, pingReq)

	if err != nil {
		errMsg := err.Error()
		return &dto.TestEndpointResponse{
			Running: false,
			Error:   &errMsg,
		}, nil
	}

	return &dto.TestEndpointResponse{
		Running: running,
	}, nil
}
