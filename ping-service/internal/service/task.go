package service

import "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"

type PingTask struct {
	Server    *domain.Server
	PrevScore int64
}
