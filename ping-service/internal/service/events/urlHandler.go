package events

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
)

type HTTPConfigEventHandler struct {
	cache       *scheduler.ServerMetaCache
	offsetStore *redis.RedisOffsetStore
}

func (h *HTTPConfigEventHandler) OnMessage(ctx context.Context, event *dto.DebeziumMessage) error {

	cfg, err := resolveConfig(event)
	if err != nil {
		return err
	}

	key := serverKeyFunc(event.TopicName, cfg.ServerID)
	isStale, err := h.isStale(ctx, key, event.ID)
	if err != nil {
		return err
	}

	if isStale {
		return nil
	}

	if err := h.handleEvent(ctx, event.Operation, cfg.ServerID); err != nil {
		return err
	}

	return h.offsetStore.SetOffset(ctx, key, event.ID)
}

func (h *HTTPConfigEventHandler) handleEvent(ctx context.Context, operation string, serverID uint) error {

	validEvents := []string{"c", "r", "u", "d"}
	if !slices.Contains(validEvents, operation) {
		return fmt.Errorf("unknown operation: %s", operation)
	}

	return h.onChanged(ctx, serverID)
}

func (h *HTTPConfigEventHandler) isStale(ctx context.Context, key string, eventID string) (bool, error) {

	offset, err := h.offsetStore.GetOffset(ctx, key)
	if err != nil {
		return false, err
	}

	return h.offsetStore.IsNewer(offset, eventID)
}

func (h *HTTPConfigEventHandler) onChanged(ctx context.Context, serverID uint) error {
	return h.cache.Delete(ctx, serverID)
}

func resolveConfig(event *dto.DebeziumMessage) (*domain.ServerHTTPConfig, error) {

	data := event.Before
	if len(data) == 0 {
		data = event.After
	}

	cfg := domain.ServerHTTPConfig{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
