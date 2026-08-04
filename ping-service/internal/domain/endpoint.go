package domain

import "time"

type K8sRuntime struct {
	LabelSelector string
	Domain        string
}

type Server struct {
	ID            uint              `json:"id"`
	Namespace     string            `json:"namespace"`
	Kind          string            `json:"kind"`
	ObjectID      string            `json:"object_id"`
	ContainerName string            `json:"container_name"`
	Interval      time.Duration     `json:"interval"`
	Timeout       time.Duration     `json:"timeout"`
	DeletedAt     *time.Time        `json:"deleted_at"`
	HTTPConfig    *ServerHTTPConfig `json:"http_config,omitempty"`

	K8s *K8sRuntime `json:"-"`
}
