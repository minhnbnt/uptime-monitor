package grpcserver

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"

	pingv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/ping/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/k8sclient"
)

type PingServer struct {
	pingv1.UnimplementedPingServiceServer
	k8sClient k8sclient.K8sClient
}

func RegisterPingServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingServer, error) {
		return &PingServer{
			k8sClient: do.MustInvoke[k8sclient.K8sClient](i),
		}, nil
	})
}

func (s *PingServer) Ping(ctx context.Context, req *pingv1.PingRequest) (*pingv1.PingResponse, error) {

	timeout := time.Duration(req.TimeoutMs) * time.Millisecond
	if timeout > 0 {

		cancel := context.CancelFunc(nil)
		ctx, cancel = context.WithTimeout(ctx, timeout)

		defer cancel()
	}

	running, err := s.k8sClient.CheckPodStatus(ctx, req.Namespace, req.Kind, req.ObjectId, req.ContainerName)
	if err != nil {
		return &pingv1.PingResponse{
			Running: false,
			Error:   fmt.Sprintf("check error: %s", err.Error()),
		}, nil
	}

	return &pingv1.PingResponse{Running: running}, nil
}
