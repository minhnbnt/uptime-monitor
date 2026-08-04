package events

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
)

type ServerEventHandler struct {
	scheduler   *scheduler.ZSetScheduleRepository
	serverCache *scheduler.ServerMetaCache
	offsetStore *redis.RedisOffsetStore
}

func serverKeyFunc(topicName string, id uint) string {
	return fmt.Sprintf("%s:%d", topicName, id)
}

func resolveServer(event *dto.DebeziumMessage) (*domain.Server, error) {

	data := event.After
	if event.Operation == "d" {
		data = event.Before
	}

	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil, errors.New("missing server payload")
	}

	server := domain.Server{}
	if err := json.Unmarshal(data, &server); err != nil {
		return nil, err
	}

	return &server, nil
}

func (e *ServerEventHandler) OnMessage(ctx context.Context, event *dto.DebeziumMessage) error {

	server, err := resolveServer(event)
	if err != nil {
		return err
	}

	key := serverKeyFunc(event.TopicName, server.ID)
	isStale, err := e.isStale(ctx, key, event.ID)
	if err != nil {
		return err
	}

	if isStale {
		return nil
	}

	if err := e.handleEvent(ctx, event.Operation, server); err != nil {
		return err
	}

	return e.offsetStore.SetOffset(ctx, key, event.ID)
}

func (e *ServerEventHandler) handleEvent(ctx context.Context, operation string, server *domain.Server) error {

	switch operation {
	case "c", "r":
		return e.onCreate(ctx, server)

	case "u":
		return e.onUpdate(ctx, server)

	case "d":
		return e.onDelete(ctx, server)

	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}
}

func (e *ServerEventHandler) isStale(ctx context.Context, key string, eventID string) (bool, error) {

	offset, err := e.offsetStore.GetOffset(ctx, key)
	if err != nil {
		return false, err
	}

	return e.offsetStore.IsNewer(offset, eventID)
}

func (e *ServerEventHandler) onCreate(ctx context.Context, server *domain.Server) error {
	return e.scheduler.Register(ctx, server)
}

func (e *ServerEventHandler) onUpdate(ctx context.Context, server *domain.Server) error {

	err := e.serverCache.Delete(ctx, server.ID)
	if err != nil {
		return err
	}

	return e.scheduler.Register(ctx, server)
}

func (e *ServerEventHandler) onDelete(ctx context.Context, server *domain.Server) error {

	if err := e.serverCache.Delete(ctx, server.ID); err != nil {
		return err
	}

	return e.scheduler.Unregister(ctx, server.ID)
}
