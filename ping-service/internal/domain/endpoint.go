package domain

import "time"

type Server struct {
	ID            uint              `json:"id"`
	Namespace     string            `json:"namespace"`
	Kind          string            `json:"kind"`
	ObjectID      string            `json:"object_id"`
	ContainerName string            `json:"container_name"`
	Interval      time.Duration     `json:"interval"`
	Timeout       time.Duration     `json:"timeout"`
	HTTPConfig    *ServerHTTPConfig `json:"http_config,omitempty"`
}
