package events

import (
	"context"
	"fmt"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis"
	"github.com/samber/lo"
)

type EventHandler interface {
	OnMessage(ctx context.Context, event *dto.DebeziumMessage) error
}

type EventMultiplexer struct {
	consumer *redis.StreamEventConsumer
	Handlers map[string]EventHandler
}

func (m *EventMultiplexer) OnMessage(ctx context.Context, event *dto.DebeziumMessage) error {

	handler, has := m.Handlers[event.TopicName]
	if !has {
		return fmt.Errorf("no handler for topic %s", event.TopicName)
	}

	if err := handler.OnMessage(ctx, event); err != nil {
		return err
	}

	return m.consumer.Ack(ctx, event)
}

func (m *EventMultiplexer) GetTopics() []string {
	return lo.Keys(m.Handlers)
}
