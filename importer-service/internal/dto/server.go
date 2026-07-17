package dto

import (
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/domain"
)

type Endpoint struct {
	URL           string
	MonitorStatus domain.ServerStatus
	Interval      time.Duration
	Timeout       time.Duration
	Method        string
	ExpectedCode  int
}

type Server struct {
	ID            uint
	Name          string
	MonitorStatus domain.ServerStatus
	Endpoint      *Endpoint
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
