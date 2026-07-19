package dto

import (
	"time"
)

type Endpoint struct {
	URL          string
	Interval     time.Duration
	Timeout      time.Duration
	Method       string
	ExpectedCode int
}

type Server struct {
	ID        uint
	Name      string
	Endpoint  *Endpoint
	CreatedAt time.Time
	UpdatedAt time.Time
}
