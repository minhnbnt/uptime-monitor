package service

import "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"

type PingTask struct {
	Server    *dto.Server
	PrevScore int64
}
