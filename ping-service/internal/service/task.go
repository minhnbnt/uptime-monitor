package service

import "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"

type PingTask struct {
	Endpoint  *domain.Endpoint
	PrevScore int64
}
