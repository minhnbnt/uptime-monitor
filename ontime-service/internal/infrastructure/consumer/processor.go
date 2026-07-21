package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type debeziumServerData struct {
	ID          uint       `json:"id"`
	CreatedByID uint       `json:"created_by_id"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

type debeziumMessage struct {
	Before *debeziumServerData `json:"before"`
	After  *debeziumServerData `json:"after"`
	Op     string              `json:"op"`
}

type messageProcessor struct {
	handler ServerOwnerHandler
	logger  *slog.Logger
}

func (p *messageProcessor) onDelete(ctx context.Context, event debeziumMessage) error {
	id, err := resolveDeletedID(event)
	if err != nil {
		return err
	}
	return p.handler.OnDelete(ctx, id)
}

func (p *messageProcessor) onUpdate(ctx context.Context, event debeziumMessage) error {
	if event.After == nil {
		return nil
	}
	return p.handler.OnUpdate(ctx, event.After.ID, event.After.CreatedByID, event.After.DeletedAt)
}

func (p *messageProcessor) onCreate(ctx context.Context, event debeziumMessage) error {
	if event.After == nil {
		return nil
	}
	return p.handler.OnCreate(ctx, event.After.ID, event.After.CreatedByID)
}

func (p *messageProcessor) ProcessMessage(ctx context.Context, msg redis.XMessage) (canAck bool) {

	raw, ok := msg.Values["value"]
	if !ok {
		p.logger.Warn("stream message missing value field", slog.String("id", msg.ID))
		return false
	}

	rawStr, ok := raw.(string)
	if !ok {
		p.logger.Warn("stream message value not string", slog.String("id", msg.ID))
		return false
	}

	event := debeziumMessage{}
	if err := json.Unmarshal([]byte(rawStr), &event); err != nil {
		p.logger.Error("stream message invalid json", slog.String("id", msg.ID), slog.Any("error", err))
		return false
	}

	switch event.Op {
	case "c", "r":
		if err := p.onCreate(ctx, event); err != nil {
			p.logger.Error("handle server",
				slog.Uint64("server_id", uint64(event.After.ID)),
				slog.String("op", event.Op),
				slog.Any("error", err),
			)
			return false
		}

	case "u":
		if err := p.onUpdate(ctx, event); err != nil {
			p.logger.Error("handle server",
				slog.Uint64("server_id", uint64(event.After.ID)),
				slog.String("op", event.Op),
				slog.Any("error", err),
			)
			return false
		}

	case "d":
		if err := p.onDelete(ctx, event); err != nil {
			p.logger.Error("handle server",
				slog.Uint64("server_id", uint64(event.Before.ID)),
				slog.String("op", event.Op),
				slog.Any("error", err),
			)
			return false
		}

	default:
		p.logger.Warn("unknown operation", slog.String("op", event.Op))
		return false
	}

	return true
}

func resolveDeletedID(event debeziumMessage) (uint, error) {
	if event.Before != nil {
		return event.Before.ID, nil
	}
	if event.After != nil {
		return event.After.ID, nil
	}
	return 0, errors.New("resolveDeletedID: event has no before or after")
}
