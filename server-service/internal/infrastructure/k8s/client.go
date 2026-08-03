package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/samber/do/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/config"
)

type Container struct {
	Name  string
	Image string
}

type K8sClient struct {
	clientSet kubernetes.Interface
}

func RegisterK8sClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*K8sClient, error) {
		clientSet := do.MustInvoke[kubernetes.Interface](i)
		return &K8sClient{clientSet: clientSet}, nil
	})
}

// EnsureNamespace creates the given namespace if it does not already exist.
// It is a no-op if the namespace already exists or the name is empty.
func (c *K8sClient) EnsureNamespace(ctx context.Context, namespace string) error {
	if namespace == "" {
		return nil
	}

	_, err := c.clientSet.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get namespace %s: %w", namespace, err)
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}
	if _, err := c.clientSet.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create namespace %s: %w", namespace, err)
	}

	return nil
}

func (c *K8sClient) CreatePod(ctx context.Context, namespace, name string, containers []Container) error {
	if err := c.EnsureNamespace(ctx, namespace); err != nil {
		return err
	}

	podSpec := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: make([]corev1.Container, 0, len(containers)),
		},
	}

	for _, ctr := range containers {
		podSpec.Spec.Containers = append(podSpec.Spec.Containers, corev1.Container{
			Name:  ctr.Name,
			Image: ctr.Image,
		})
	}

	_, err := c.clientSet.CoreV1().Pods(namespace).Create(ctx, podSpec, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create pod %s/%s: %w", namespace, name, err)
	}

	return nil
}

func (c *K8sClient) DeletePod(ctx context.Context, namespace, name string) error {
	err := c.clientSet.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete pod %s/%s: %w", namespace, name, err)
	}

	return nil
}

func RegisterClientset(i do.Injector) {
	do.Provide(i, func(i do.Injector) (kubernetes.Interface, error) {

		logger := do.MustInvoke[*slog.Logger](i)
		cfg := do.MustInvoke[*config.Config](i)

		config, err := rest.InClusterConfig()
		if err != nil {
			logger.Info("in-cluster config unavailable, falling back to kubeconfig", slog.Any("error", err))
			config, err = clientcmd.BuildConfigFromFlags("", resolveKubeconfig(cfg.Kubeconfig))
			if err != nil {
				return nil, fmt.Errorf("k8s config: in-cluster and kubeconfig both failed: %w", err)
			}
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, fmt.Errorf("k8s clientset: %w", err)
		}

		return clientset, nil
	})
}

func resolveKubeconfig(cfgPath string) string {
	if cfgPath != "" {
		return cfgPath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}
