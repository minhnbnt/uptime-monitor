package dto

import "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"

// K8sObjectKey identifies a k8s object by its cluster-unique coordinates.
type K8sObjectKey struct {
	Namespace string
	Kind      string
	ObjectID  string
}

type CheckParams struct {
	K8sObjectCheckParams
	HTTPCheckParams *HTTPCheckParams
}

type K8sObjectCheckParams struct {
	K8sObjectKey
	ContainerName string // "" means no container check

	K8s *domain.K8sRuntime // cached k8s-derived values, nil = resolve live
}

// NewK8sObjectCheckParams maps a domain.Server to check params; Server already
// carries every field K8sObjectCheckParams needs.
func NewK8sObjectCheckParams(sv *domain.Server) *K8sObjectCheckParams {
	return &K8sObjectCheckParams{
		K8sObjectKey: K8sObjectKey{
			Namespace: sv.Namespace,
			Kind:      sv.Kind,
			ObjectID:  sv.ObjectID,
		},
		ContainerName: sv.ContainerName,
		K8s:           sv.K8s,
	}
}

type HTTPCheckParams struct {
	Method        string
	Port          int
	EndpointPath  string
	ExpectedCode  int
	BodyCheckExpr string // "" means no body check
}
