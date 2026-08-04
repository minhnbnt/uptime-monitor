package k8sclient

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1Types "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
)

type k8sWorkloadChecker struct {
	labelSelector *k8sLabelSelector
	client        corev1Types.CoreV1Interface
}

func newWorkloadChecker(clientSet kubernetes.Interface, labelSelector *k8sLabelSelector) *k8sWorkloadChecker {
	return &k8sWorkloadChecker{
		client:        clientSet.CoreV1(),
		labelSelector: labelSelector,
	}
}

func IsWorkloadKind(kind string) bool {
	switch kind {
	case "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet":
		return true

	default:
		return false
	}
}

func (c *k8sWorkloadChecker) CheckObject(ctx context.Context, params *dto.K8sObjectCheckParams) (bool, error) {

	switch params.Kind {
	case "Pod":
		return c.checkPod(ctx, params)

	default:
		if !IsWorkloadKind(params.Kind) {
			return false, fmt.Errorf("unsupported kind: %s", params.Kind)
		}
		return c.checkWorkload(ctx, params)
	}
}

func (c *k8sWorkloadChecker) checkWorkload(ctx context.Context, params *dto.K8sObjectCheckParams) (bool, error) {

	selector := ""
	if params.K8s != nil {
		selector = params.K8s.LabelSelector
	}

	if selector == "" {
		selector, err := c.labelSelector.getWorkloadLabelSelector(
			ctx, params.Namespace,
			params.Kind, params.ObjectID,
		)

		if err != nil {
			return false, err
		}
		if selector == "" {
			return false, nil
		}
	}

	option := metav1.ListOptions{LabelSelector: selector}
	pods, err := c.client.Pods(params.Namespace).List(ctx, option)

	if err != nil {
		return false, fmt.Errorf("list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return false, nil
	}

	for i := range pods.Items {

		containerName := params.ContainerName

		if isContainerRunning(&pods.Items[i], containerName) {
			return true, nil
		}

		if isPodRunning(&pods.Items[i]) {
			return true, nil
		}
	}

	return false, nil
}

func (c *k8sWorkloadChecker) checkPod(ctx context.Context, params *dto.K8sObjectCheckParams) (bool, error) {

	pod, err := c.client.Pods(params.Namespace).Get(ctx, params.ObjectID, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("pod not found: %w", err)
	}

	if params.ContainerName == "" {
		return isPodRunning(pod), nil
	}

	return isContainerRunning(pod, params.ContainerName), nil
}

func isContainerRunning(pod *corev1.Pod, containerName string) bool {

	statuses := pod.Status.ContainerStatuses
	if len(statuses) == 0 {
		return false
	}

	target := lo.Filter(statuses, func(status corev1.ContainerStatus, _ int) bool {
		return status.Name == containerName
	})

	if len(target) != 1 {
		return false
	}

	return target[0].Ready
}

func isPodRunning(pod *corev1.Pod) bool {

	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	statuses := pod.Status.ContainerStatuses
	if len(statuses) == 0 {
		return false
	}

	for _, s := range statuses {
		if !s.Ready {
			return false
		}
	}

	return true
}
