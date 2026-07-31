package events

import (
	"context"
	"encoding/json"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
)

type ServerEventHandler struct {
	scheduler   *scheduler.ZSetScheduleRepository
	serverCache *scheduler.ServerMetaCache
}

func (e *ServerEventHandler) OnMessage(ctx context.Context, event *dto.DebeziumMessage) error {

	switch event.Operation {
	case "c", "r":
		return e.onCreate(ctx, event)

	case "u":
		return e.onUpdate(ctx, event)

	case "d":
		return e.onDelete(ctx, event)

	default:
		return nil
	}
}

func (e *ServerEventHandler) onCreate(ctx context.Context, event *dto.DebeziumMessage) error {

	server := domain.Server{}
	if err := json.Unmarshal(event.After, &server); err != nil {
		return err
	}

	return e.scheduler.Register(ctx, &server)
}

func (e *ServerEventHandler) onUpdate(ctx context.Context, event *dto.DebeziumMessage) error {

	server := domain.Server{}
	if err := json.Unmarshal(event.After, &server); err != nil {
		return err
	}

	err := e.serverCache.Delete(ctx, server.ID)
	if err != nil {
		return err
	}

	return e.scheduler.Register(ctx, &server)
}

func (e *ServerEventHandler) onDelete(ctx context.Context, event *dto.DebeziumMessage) error {

	data := event.Before
	if len(data) == 0 {
		data = event.After
	}

	server := domain.Server{}
	if err := json.Unmarshal(data, &server); err != nil {
		return err
	}

	err := e.serverCache.Delete(ctx, server.ID)
	if err != nil {
		return err
	}

	return e.scheduler.Unregister(ctx, server.ID)
}
