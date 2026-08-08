package dto

import (
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

// Server is the service-layer shape of a domain.Server with check params
// prebuilt, so downstream consumers never re-map from domain.
type Server struct {
	ID       uint
	Interval time.Duration
	Timeout  time.Duration

	K8sObjectCheckParams
	HTTPCheckParams *HTTPCheckParams
}

func NewServer(sv *domain.Server) *Server {

	s := &Server{
		ID:       sv.ID,
		Interval: sv.Interval,
		Timeout:  sv.Timeout,
	}

	s.K8sObjectCheckParams = K8sObjectCheckParams{
		K8sObjectKey: K8sObjectKey{
			Namespace: sv.Namespace,
			Kind:      sv.Kind,
			ObjectID:  sv.ObjectID,
		},
		ContainerName: sv.ContainerName,
		K8s:           sv.K8s,
	}

	if cfg := sv.HTTPConfig; cfg != nil {
		s.HTTPCheckParams = &HTTPCheckParams{
			Method:        cfg.Method,
			Port:          cfg.Port,
			EndpointPath:  cfg.EndpointPath,
			ExpectedCode:  cfg.ExpectedCode,
			BodyCheckExpr: cfg.BodyCheckExpr,
		}
	}

	return s
}
