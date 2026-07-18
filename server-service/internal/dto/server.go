package dto

import (
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
)

type Endpoint struct {
	ID           uint
	URL          string
	Interval     time.Duration
	Timeout      time.Duration
	Method       string
	ExpectedCode int
}

func EndpointFromDomain(e *domain.Endpoint) *Endpoint {

	if e == nil {
		return nil
	}

	return &Endpoint{
		ID:           e.ID,
		URL:          e.URL,
		Interval:     e.Interval,
		Timeout:      e.Timeout,
		Method:       e.Method,
		ExpectedCode: e.ExpectedCode,
	}
}

type Server struct {
	ID           uint
	Name         string
	CreatedByID  uint
	Endpoint     *Endpoint
	MonitorStatus domain.ServerStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func ServerFromDomain(s domain.Server) Server {
	return Server{
		ID:          s.ID,
		Name:        s.Name,
		CreatedByID: s.CreatedByID,
		Endpoint:    EndpointFromDomain(s.Endpoint),
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

type CheckMethodType string

const (
	CheckMethodPull CheckMethodType = "pull"
	CheckMethodPush CheckMethodType = "push"
)

type CreateServerRequest struct {
	Name string
}

type UpdateServerRequest struct {
	Name *string
}

type SetCheckMethodRequest struct {
	URL          string
	Method       CheckMethodType
	HTTPMethod   string
	Interval     time.Duration
	Timeout      time.Duration
	ExpectedCode int
}

type TestEndpointRequest struct {
	URL          string
	Method       string
	Timeout      time.Duration
	ExpectedCode int
}

type TestEndpointResponse struct {
	Success    bool
	StatusCode int
	Error      *string
}
