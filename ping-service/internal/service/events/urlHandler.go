package events

import (
	"context"
	"encoding/json"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
)

type HTTPConfigEventHandler struct {
	cache *scheduler.ServerMetaCache
}

func (h *HTTPConfigEventHandler) OnMessage(ctx context.Context, event *dto.DebeziumMessage) error {

	switch event.Operation {
	case "c", "r":
		return h.onCreate(ctx, event)

	case "u":
		return h.onUpdate(ctx, event)

	case "d":
		return h.onDelete(ctx, event)

	default:
		return nil
	}
}

func (h *HTTPConfigEventHandler) onCreate(ctx context.Context, event *dto.DebeziumMessage) error {

	cfg := domain.ServerHTTPConfig{}
	if err := json.Unmarshal(event.After, &cfg); err != nil {
		return err
	}

	return h.cache.Delete(ctx, cfg.ServerID)
}

func (h *HTTPConfigEventHandler) onUpdate(ctx context.Context, event *dto.DebeziumMessage) error {

	cfg := domain.ServerHTTPConfig{}
	if err := json.Unmarshal(event.Before, &cfg); err != nil {
		return err
	}

	return h.cache.Delete(ctx, cfg.ServerID)
}

func (h *HTTPConfigEventHandler) onDelete(ctx context.Context, event *dto.DebeziumMessage) error {

	data := event.Before
	if len(data) == 0 {
		data = event.After
	}

	cfg := domain.ServerHTTPConfig{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	return h.cache.Delete(ctx, cfg.ServerID)
}
