package dto

import (
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
)

type Endpoint struct {
	URL           string
	MonitorStatus domain.ServerStatus
	Interval      time.Duration
	Timeout       time.Duration
	Method        string
	ExpectedCode  int
}

func EndpointFromDomain(e *domain.Endpoint) *Endpoint {
	if e == nil {
		return nil
	}
	return &Endpoint{
		URL:           e.URL,
		MonitorStatus: e.MonitorStatus,
		Interval:      e.Interval,
		Timeout:       e.Timeout,
		Method:        e.Method,
		ExpectedCode:  e.ExpectedCode,
	}
}

type Server struct {
	ID            uint
	Name          string
	CreatedByID   uint
	MonitorStatus domain.ServerStatus
	Endpoint      *Endpoint
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func ServerFromDomain(s domain.Server) Server {
	ms := domain.StatusOff
	if s.Endpoint != nil {
		ms = s.Endpoint.MonitorStatus
	}
	return Server{
		ID:            s.ID,
		Name:          s.Name,
		CreatedByID:   s.CreatedByID,
		MonitorStatus: ms,
		Endpoint:      EndpointFromDomain(s.Endpoint),
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
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
