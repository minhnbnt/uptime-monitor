package k8sclient

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *k8sClient) checkHTTPDNS(ctx context.Context, params PingCheck) (bool, error) {

	url, err := c.getURL(ctx, params)
	if err != nil {
		return false, err
	}

	resp, err := c.pingClient.Ping(ctx, 0, params.Method, url)
	if err != nil {
		return false, err
	}

	return checkOK(resp.StatusCode, params.ExpectedCode), nil
}

func (c *k8sClient) getURL(ctx context.Context, params PingCheck) (string, error) {

	switch params.Kind {
	case "Service":
		return fmt.Sprintf(
			"http://%s.%s.svc.cluster.local:%d/%s",
			params.ObjectID,
			params.Namespace,
			params.Port,
			params.EndpointPath,
		), nil

	case "Pod":
		return c.resolvePodURL(ctx, params)

	case "StatefulSet":
		return fmt.Sprintf(
			"http://%s-0.%s.%s.svc.cluster.local:%d/%s",
			params.ObjectID,
			params.ObjectID,
			params.Namespace,
			params.Port,
			params.EndpointPath,
		), nil

	default:
		return "", fmt.Errorf("http-dns not supported for kind: %s", params.Kind)
	}
}

func (c *k8sClient) resolvePodURL(ctx context.Context, params PingCheck) (string, error) {

	pod, err := c.clientset.CoreV1().Pods(params.Namespace).Get(ctx, params.ObjectID, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	if pod.Status.PodIP == "" {
		return "", fmt.Errorf("pod ip not found")
	}

	return fmt.Sprintf("http://%s:%d/%s", pod.Status.PodIP, params.Port, params.EndpointPath), nil
}

func checkOK(statusCode, expectedCode int) bool {

	if expectedCode > 0 {
		return statusCode == expectedCode
	}

	return statusCode >= 200 && statusCode < 300
}
