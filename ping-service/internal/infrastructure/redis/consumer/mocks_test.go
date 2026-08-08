package consumer

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
)

type mockServerEventHandler struct {
	onMessageFn func(ctx context.Context, event *dto.DebeziumMessage) error
}

func (m *mockServerEventHandler) OnMessage(ctx context.Context, event *dto.DebeziumMessage) error {
	if m.onMessageFn == nil {
		return nil
	}
	return m.onMessageFn(ctx, event)
}

var _ ServerEventHandler = (*mockServerEventHandler)(nil)
