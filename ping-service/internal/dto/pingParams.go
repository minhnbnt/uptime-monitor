package dto

import "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"

type CheckParams struct {
	K8sObjectCheckParams
	HTTPCheckParams *HTTPCheckParams
}

type K8sObjectCheckParams struct {
	Namespace     string
	Kind          string
	ObjectID      string
	ContainerName string // "" means no container check

	K8s *domain.K8sRuntime // cached k8s-derived values, nil = resolve live
}

type HTTPCheckParams struct {
	Method        string
	Port          int
	EndpointPath  string
	ExpectedCode  int
	BodyCheckExpr string // "" means no body check
}
