package handler

import (
	"context"

	ontimedto "github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
)

type OntimeService interface {
	ListServersWithOntime(ctx context.Context, createdByID uint, page, perPage int) ([]ontimedto.ServerWithOntime, int64, error)
	GetServerWithOntime(ctx context.Context, serverID uint) (*ontimedto.ServerWithOntime, error)
}
