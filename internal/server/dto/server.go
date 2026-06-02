package dto

import (
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
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

type CheckMethodType string

const (
	CheckMethodPull CheckMethodType = "pull"
	CheckMethodPush CheckMethodType = "push"
)

type CreateServerRequest struct {
	Name string `validate:"required,min=1,max=255"`
}

type UpdateServerRequest struct {
	Name   *string        `validate:"omitempty,min=1,max=255"`
	Status *domain.Status `validate:"omitempty,oneof=active paused"`
}

type SetCheckMethodRequest struct {
	URL          string          `validate:"required,url"`
	Method       CheckMethodType `validate:"required,oneof=push pull"`
	HTTPMethod   string          `validate:"required,oneof=GET POST PUT DELETE PATCH HEAD OPTIONS CONNECT TRACE"`
	Interval     time.Duration   `validate:"required,gt=0"`
	Timeout      time.Duration   `validate:"required,gt=0"`
	ExpectedCode int             `validate:"min=100,max=599"`
}
