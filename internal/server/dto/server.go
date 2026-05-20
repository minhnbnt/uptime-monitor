package dto

import (
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/server/domain"
)

type Endpoint struct {
	URL          string
	Status       domain.Status
	Interval     time.Duration
	Timeout      time.Duration
	Method       string
	ExpectedCode int
}

type Server struct {
	ID        uint
	Name      string
	Status    domain.Status
	Endpoint  *Endpoint
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateServerRequest struct {
	Name string
}

type UpdateServerRequest struct {
	Name   *string
	Status *domain.Status
}
