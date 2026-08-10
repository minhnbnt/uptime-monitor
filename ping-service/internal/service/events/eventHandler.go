package events

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
)

type EventHandler interface {
	OnMessage(ctx context.Context, event *dto.DebeziumMessage) error
}

type AckClient interface {
	Ack(ctx context.Context, event *dto.DebeziumMessage) error
}

type EventMultiplexer struct {
	Handlers  map[string]EventHandler
	AckClient AckClient
}

func (m *EventMultiplexer) OnMessage(ctx context.Context, event *dto.DebeziumMessage) error {

	handler, has := m.Handlers[event.TopicName]
	if !has {
		return fmt.Errorf("no handler for topic %s", event.TopicName)
	}

	if err := handler.OnMessage(ctx, event); err != nil {
		return err
	}

	return m.AckClient.Ack(ctx, event)
}

func (m *EventMultiplexer) GetTopics() []string {
	return lo.Keys(m.Handlers)
}
