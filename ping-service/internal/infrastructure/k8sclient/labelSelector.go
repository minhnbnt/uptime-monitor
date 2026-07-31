package k8sclient

import (
	"context"
	"fmt"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type k8sLabelSelector struct {
	logger    *slog.Logger
	clientset kubernetes.Interface
}

func newLabelSelector(logger *slog.Logger, clientset kubernetes.Interface) *k8sLabelSelector {
	return &k8sLabelSelector{
		logger:    logger,
		clientset: clientset,
	}
}

func (c *k8sLabelSelector) getWorkloadLabelSelector(ctx context.Context, kind, namespace, name string) (string, error) {

	switch kind {
	case "Deployment":
		return c.getDeploymentLabelSelector(ctx, namespace, name)

	case "StatefulSet":
		return c.getStatefulSetLabelSelector(ctx, namespace, name)

	case "DaemonSet":
		return c.getDaemonSetLabelSelector(ctx, namespace, name)

	case "ReplicaSet":
		return c.getReplicaSetLabelSelector(ctx, namespace, name)

	default:
		return "", fmt.Errorf("unsupported kind: %s", kind)
	}
}

func (c *k8sLabelSelector) getDeploymentLabelSelector(ctx context.Context, namespace, name string) (string, error) {

	deployment, err := c.clientset.AppsV1().
		Deployments(namespace).
		Get(ctx, name, metav1.GetOptions{})

	if err != nil {
		return "", err
	}

	return metav1.FormatLabelSelector(deployment.Spec.Selector), nil
}

func (c *k8sLabelSelector) getStatefulSetLabelSelector(ctx context.Context, namespace, name string) (string, error) {

	statefulSet, err := c.clientset.AppsV1().
		StatefulSets(namespace).
		Get(ctx, name, metav1.GetOptions{})

	if err != nil {
		return "", err
	}

	return metav1.FormatLabelSelector(statefulSet.Spec.Selector), nil
}

func (c *k8sLabelSelector) getDaemonSetLabelSelector(ctx context.Context, namespace, name string) (string, error) {

	daemonSet, err := c.clientset.AppsV1().
		DaemonSets(namespace).
		Get(ctx, name, metav1.GetOptions{})

	if err != nil {
		return "", err
	}

	return metav1.FormatLabelSelector(daemonSet.Spec.Selector), nil
}

func (c *k8sLabelSelector) getReplicaSetLabelSelector(ctx context.Context, namespace, name string) (string, error) {

	replicaSet, err := c.clientset.AppsV1().
		ReplicaSets(namespace).
		Get(ctx, name, metav1.GetOptions{})

	if err != nil {
		return "", err
	}

	return metav1.FormatLabelSelector(replicaSet.Spec.Selector), nil
}
