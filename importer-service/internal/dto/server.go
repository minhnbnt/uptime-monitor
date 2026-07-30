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
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
