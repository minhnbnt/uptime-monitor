package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type Server struct {
	ID        uuid.UUID
	Name      string
	URL       string
	Status    domain.Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateServerRequest struct {
	Name string
	URL  string
}

type UpdateServerRequest struct {
	Name   *string
	URL    *string
	Status *domain.Status
}
