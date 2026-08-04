package k8sclient

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"
	"k8s.io/client-go/kubernetes"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
)

type K8sClient struct {
	workloadChecker *k8sWorkloadChecker
	domainResolver  *k8sDomainNameResolver
}

func RegisterK8sClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*K8sClient, error) {

		logger := do.MustInvoke[*slog.Logger](i)
		clientSet := do.MustInvoke[kubernetes.Interface](i)

		labelSelector := newLabelSelector(logger, clientSet)

		return &K8sClient{
			workloadChecker: newWorkloadChecker(clientSet, labelSelector),
			domainResolver:  newDomainResolver(clientSet),
		}, nil
	})
}

func (c *K8sClient) CheckObjectStatus(ctx context.Context, params *dto.K8sObjectCheckParams) (bool, error) {
	return c.workloadChecker.CheckObject(ctx, params)
}

func (c *K8sClient) ResolveDomainName(ctx context.Context, params *dto.K8sObjectCheckParams) (string, error) {
	return c.domainResolver.ResolveDomainName(ctx, params)
}

func (c *K8sClient) ResolveLabelSelector(ctx context.Context, params *dto.K8sObjectCheckParams) (string, error) {
	return c.workloadChecker.labelSelector.getWorkloadLabelSelector(
		ctx, params.Namespace, params.Kind, params.ObjectID,
	)
}
