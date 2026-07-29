package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/samber/do/v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func RegisterK8sClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (kubernetes.Interface, error) {
		logger := do.MustInvoke[*slog.Logger](i)

		config, err := rest.InClusterConfig()
		if err != nil {
			logger.Info("in-cluster config unavailable, falling back to kubeconfig", slog.Any("error", err))
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfigOrDefault())
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

func kubeconfigOrDefault() string {
	if v := os.Getenv("KUBECONFIG"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}
