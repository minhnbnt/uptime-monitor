package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
)

type OntimeService interface {
	ListServersWithOntime(ctx context.Context, createdByID uint, page, perPage int) ([]dto.ServerOntime, error)
	GetServerWithOntime(ctx context.Context, serverID uint, userID uint) (*dto.ServerOntime, error)
}
