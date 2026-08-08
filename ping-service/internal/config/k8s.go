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
		cfg := do.MustInvoke[*Config](i)

		config, err := rest.InClusterConfig()
		if err != nil {
			logger.Info("in-cluster config unavailable, falling back to kubeconfig", slog.Any("error", err))
			config, err = clientcmd.BuildConfigFromFlags("", resolveKubeconfig(cfg.Kubeconfig))
			if err != nil {
				return nil, fmt.Errorf("k8s config: in-cluster and kubeconfig both failed: %w", err)
			}
		}

		// 10k k8s API requests per 30s ping cycle = 334 req/s sustained.
		// QPS is the token fill rate, Burst the bucket capacity; oversize
		// both so the workloop never blocks on client-side throttling.
		config.QPS = cfg.K8s.QPS
		config.Burst = cfg.K8s.Burst

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
