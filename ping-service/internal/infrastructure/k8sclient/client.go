package k8sclient

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
)

type PingCheck struct {
	Namespace     string
	Kind          string
	ObjectID      string
	ContainerName string
	PingType      uint
	Port          int
	EndpointPath  string
	ExpectedCode  int
	BodyCheckExpr *string
}

type K8sClient interface {
	CheckPodStatus(ctx context.Context, params PingCheck) (bool, error)
}

type k8sClient struct {
	clientset   kubernetes.Interface
	logger      *slog.Logger
	httpClient  *http.Client
	bodyChecker *infrastructure.BodyChecker
}

func RegisterK8sClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (K8sClient, error) {
		return &k8sClient{
			clientset:   do.MustInvoke[kubernetes.Interface](i),
			logger:      do.MustInvoke[*slog.Logger](i),
			httpClient:  &http.Client{Timeout: 10 * time.Second},
			bodyChecker: do.MustInvoke[*infrastructure.BodyChecker](i),
		}, nil
	})
}

func (c *k8sClient) CheckPodStatus(ctx context.Context, params PingCheck) (bool, error) {
	if params.PingType == 1 {
		return c.checkHTTPDNS(ctx, params.Namespace, params.Kind, params.ObjectID, params.Port, params.EndpointPath, params.ExpectedCode, params.BodyCheckExpr)
	}
	switch params.Kind {
	case "Pod":
		return c.checkPod(ctx, params.Namespace, params.ObjectID, params.ContainerName)
	case "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet":
		return c.checkWorkload(ctx, params.Namespace, params.Kind, params.ObjectID, params.ContainerName)
	default:
		return false, fmt.Errorf("unsupported kind: %s", params.Kind)
	}
}

func (c *k8sClient) checkHTTPDNS(ctx context.Context, namespace, kind, objectID string, port int, endpointPath string, expectedCode int, bodyCheckExpr *string) (bool, error) {
	switch kind {
	case "Service":
		return c.checkService(ctx, namespace, objectID, port, endpointPath, expectedCode, bodyCheckExpr)
	case "Pod":
		return c.checkPodHTTP(ctx, namespace, objectID, port, endpointPath, expectedCode, bodyCheckExpr)
	case "StatefulSet":
		return c.checkStatefulSetHTTP(ctx, namespace, objectID, port, endpointPath, expectedCode, bodyCheckExpr)
	default:
		return false, fmt.Errorf("http-dns not supported for kind: %s", kind)
	}
}

func (c *k8sClient) checkService(ctx context.Context, namespace, name string, port int, endpointPath string, expectedCode int, bodyCheckExpr *string) (bool, error) {
	target := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d/%s", name, namespace, port, endpointPath)
	return c.httpGET(ctx, target, expectedCode, bodyCheckExpr)
}

func (c *k8sClient) checkPodHTTP(ctx context.Context, namespace, name string, port int, endpointPath string, expectedCode int, bodyCheckExpr *string) (bool, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		c.logger.Debug("pod not found for http-dns", slog.String("namespace", namespace), slog.String("name", name), slog.Any("error", err))
		return false, nil
	}
	if pod.Status.PodIP == "" {
		return false, nil
	}
	target := fmt.Sprintf("http://%s:%d/%s", pod.Status.PodIP, port, endpointPath)
	return c.httpGET(ctx, target, expectedCode, bodyCheckExpr)
}

func (c *k8sClient) checkStatefulSetHTTP(ctx context.Context, namespace, name string, port int, endpointPath string, expectedCode int, bodyCheckExpr *string) (bool, error) {
	target := fmt.Sprintf("http://%s-0.%s.%s.svc.cluster.local:%d/%s", name, name, namespace, port, endpointPath)
	return c.httpGET(ctx, target, expectedCode, bodyCheckExpr)
}

func (c *k8sClient) httpGET(ctx context.Context, url string, expectedCode int, bodyCheckExpr *string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false, fmt.Errorf("read body: %w", err)
	}

	if expectedCode > 0 && resp.StatusCode != expectedCode {
		return false, nil
	}

	if bodyCheckExpr != nil && *bodyCheckExpr != "" {
		ok, err := c.bodyChecker.Check(string(body), *bodyCheckExpr)
		if err != nil || !ok {
			return false, nil
		}
	}

	if expectedCode == 0 && bodyCheckExpr == nil {
		return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
	}

	return true, nil
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
