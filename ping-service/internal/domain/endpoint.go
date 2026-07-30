package domain

import "time"

type Server struct {
	ID            uint          `json:"id"`
	Namespace     string        `json:"namespace"`
	Kind          string        `json:"kind"`
	ObjectID      string        `json:"object_id"`
	ContainerName string        `json:"container_name"`
	Interval      time.Duration `json:"interval"`
	Timeout       time.Duration `json:"timeout"`
	PingType      uint          `json:"ping_type"`
	Method        string        `json:"method"`
	Port          int           `json:"port"`
	EndpointPath  string        `json:"endpoint_path"`
	ExpectedCode  int           `json:"expected_code"`
	BodyCheckExpr *string       `json:"body_check_expr"`
}

// Endpoint is a type alias for backward compatibility with dead code.
type Endpoint = Server
