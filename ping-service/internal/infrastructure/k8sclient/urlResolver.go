package k8sclient

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1Types "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
)

type k8sDomainNameResolver struct {
	client corev1Types.CoreV1Interface
}

func newDomainResolver(clientSet kubernetes.Interface) *k8sDomainNameResolver {
	return &k8sDomainNameResolver{client: clientSet.CoreV1()}
}

func (r *k8sDomainNameResolver) ResolveDomainName(ctx context.Context, params *dto.K8sObjectCheckParams) (string, error) {

	if params.K8s != nil && params.K8s.Domain != "" {
		return params.K8s.Domain, nil
	}

	switch params.Kind {
	case "Service":
		return fmt.Sprintf(
			"%s.%s.svc.cluster.local",
			params.ObjectID,
			params.Namespace,
		), nil

	case "StatefulSet":
		return fmt.Sprintf(
			"%s-0.%s.%s.svc.cluster.local",
			params.ObjectID,
			params.ObjectID,
			params.Namespace,
		), nil

	case "Pod":
		return r.resolvePodURL(ctx, params)

	default:
		return "", fmt.Errorf("http-dns not supported for kind: %s", params.Kind)
	}
}

func (r *k8sDomainNameResolver) resolvePodURL(ctx context.Context, params *dto.K8sObjectCheckParams) (string, error) {

	pod, err := r.client.Pods(params.Namespace).Get(ctx, params.ObjectID, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	if pod.Status.PodIP == "" {
		return "", fmt.Errorf("pod ip not found")
	}

	return pod.Status.PodIP, nil
}
