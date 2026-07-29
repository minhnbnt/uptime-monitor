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

	// Deprecated: kept for dead code compatibility (responseChecker.go).
	ExpectedCode  int     `json:"-"`
	BodyCheckExpr *string `json:"-"`
}

// Endpoint is a type alias for backward compatibility with dead code.
type Endpoint = Server
