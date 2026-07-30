package k8sclient

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/do/v2"
	"k8s.io/client-go/kubernetes"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
)

type PingCheck struct {
	Namespace     string
	Kind          string
	ObjectID      string
	ContainerName string
	PingType      uint
	Method        string
	Port          int
	EndpointPath  string
	ExpectedCode  int
	BodyCheckExpr *string
}

type K8sClient interface {
	CheckPodStatus(ctx context.Context, params PingCheck) (bool, error)
}

type k8sClient struct {
	clientset  kubernetes.Interface
	logger     *slog.Logger
	pingClient *infrastructure.PingClient
}

func RegisterK8sClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (K8sClient, error) {
		return &k8sClient{
			clientset:  do.MustInvoke[kubernetes.Interface](i),
			logger:     do.MustInvoke[*slog.Logger](i),
			pingClient: do.MustInvoke[*infrastructure.PingClient](i),
		}, nil
	})
}

func (c *k8sClient) CheckPodStatus(ctx context.Context, params PingCheck) (bool, error) {
	if params.PingType == 1 {
		return c.checkHTTPDNS(ctx, params)
	}

	switch params.Kind {
	case "Pod":
		return c.checkPod(ctx, params.Namespace, params.ObjectID, params.ContainerName)
	case "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet":
		return c.checkWorkload(ctx, params)
	default:
		return false, fmt.Errorf("unsupported kind: %s", params.Kind)
	}
}
