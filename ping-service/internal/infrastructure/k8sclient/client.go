package k8sclient

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sClient interface {
	CheckPodStatus(ctx context.Context, namespace, kind, objectID, containerName string) (bool, error)
}

type k8sClient struct {
	clientset kubernetes.Interface
	logger    *slog.Logger
}

func RegisterK8sClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (K8sClient, error) {
		return &k8sClient{
			clientset: do.MustInvoke[kubernetes.Interface](i),
			logger:    do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (c *k8sClient) CheckPodStatus(ctx context.Context, namespace, kind, objectID, containerName string) (bool, error) {
	switch kind {
	case "Pod":
		return c.checkPod(ctx, namespace, objectID, containerName)
	case "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet":
		return c.checkWorkload(ctx, namespace, kind, objectID, containerName)
	default:
		return false, fmt.Errorf("unsupported kind: %s", kind)
	}
}

func (c *k8sClient) checkPod(ctx context.Context, namespace, name, containerName string) (bool, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		c.logger.Debug("pod not found", slog.String("namespace", namespace), slog.String("name", name), slog.Any("error", err))
		return false, nil
	}
	return isPodRunning(pod, containerName), nil
}

func (c *k8sClient) checkWorkload(ctx context.Context, namespace, kind, name, containerName string) (bool, error) {

	selector, err := c.getWorkloadLabelSelector(ctx, namespace, kind, name)
	if err != nil {
		return false, err
	}
	if selector == "" {
		return false, nil
	}

	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return false, fmt.Errorf("list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return false, nil
	}

	for i := range pods.Items {
		if isPodRunning(&pods.Items[i], containerName) {
			return true, nil
		}
	}
	return false, nil
}

func (c *k8sClient) getWorkloadLabelSelector(ctx context.Context, namespace, kind, name string) (string, error) {
	switch kind {
	case "Deployment":
		dep, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.logger.Debug("deployment not found", slog.String("namespace", namespace), slog.String("name", name), slog.Any("error", err))
			return "", nil
		}
		return metav1.FormatLabelSelector(dep.Spec.Selector), nil

	case "StatefulSet":
		sts, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.logger.Debug("statefulset not found", slog.String("namespace", namespace), slog.String("name", name), slog.Any("error", err))
			return "", nil
		}
		return metav1.FormatLabelSelector(sts.Spec.Selector), nil
	case "DaemonSet":
		ds, err := c.clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.logger.Debug("daemonset not found", slog.String("namespace", namespace), slog.String("name", name), slog.Any("error", err))
			return "", nil
		}
		return metav1.FormatLabelSelector(ds.Spec.Selector), nil
	case "ReplicaSet":
		rs, err := c.clientset.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.logger.Debug("replicaset not found", slog.String("namespace", namespace), slog.String("name", name), slog.Any("error", err))
			return "", nil
		}
		return metav1.FormatLabelSelector(rs.Spec.Selector), nil
	default:
		return "", fmt.Errorf("unsupported workload kind: %s", kind)
	}
}

func isPodRunning(pod *corev1.Pod, containerName string) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	statuses := pod.Status.ContainerStatuses
	if len(statuses) == 0 {
		return false
	}

	if containerName != "" {
		target := lo.Filter(statuses, func(status corev1.ContainerStatus, _ int) bool {
			return status.Name == containerName
		})
		if len(target) != 1 {
			return false
		}
		return target[0].Ready
	}

	for _, s := range statuses {
		if !s.Ready {
			return false
		}
	}
	return true
}
