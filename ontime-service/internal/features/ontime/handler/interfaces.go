package handler

import (
	"context"

	ontimedto "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/features/ontime/dto"
)

type OntimeService interface {
	ListServersWithOntime(ctx context.Context, createdByID uint, page, perPage int) ([]ontimedto.ServerOntime, error)
	GetServerWithOntime(ctx context.Context, serverID uint, userID uint) (*ontimedto.ServerOntime, error)
}
