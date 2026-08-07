package dto

import (
	"time"
)

type Server struct {
	ID            uint
	Name          string
	Namespace     string
	Kind          string
	ObjectID      string
	ContainerName string
	Interval      time.Duration
	Timeout       time.Duration
	HTTPConfig    *HTTPConfig
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type HTTPConfig struct {
	Port          int
	EndpointPath  string
	ExpectedCode  int
	BodyCheckExpr string
	Method        string
}
