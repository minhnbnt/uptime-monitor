package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
)

type messageProcessor struct {
	handler ServerEventHandler
	logger  *slog.Logger
}

func (p *messageProcessor) ProcessMessage(ctx context.Context, streamKey string, msg redis.XMessage) {

	raw, ok := msg.Values["value"]
	if !ok {
		p.logger.Warn("stream message missing value field", slog.String("id", msg.ID))
		return
	}

	rawStr, ok := raw.(string)
	if !ok {

		p.logger.Warn(
			"stream message value not string",
			slog.String("id", msg.ID),
		)

		return
	}

	event := dto.DebeziumMessage{ID: msg.ID, TopicName: streamKey}
	if err := json.Unmarshal([]byte(rawStr), &event); err != nil {

		p.logger.Error(
			"stream message invalid json",
			slog.String("id", msg.ID),
			slog.Any("error", err),
		)

		return
	}

	if err := p.handler.OnMessage(ctx, &event); err != nil {
		p.logger.Error(
			"handle event",
			slog.String("id", msg.ID),
			slog.String("topic", streamKey),
			slog.Any("error", err),
		)
	}
}
