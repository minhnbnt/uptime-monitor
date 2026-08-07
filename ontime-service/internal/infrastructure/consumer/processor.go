package consumer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

type messageProcessor struct {
	handler ServerOwnerHandler
	logger  *slog.Logger
	offsets *RedisOffsetStore
}

func (p *messageProcessor) onDelete(ctx context.Context, event debeziumMessage) error {

	id, err := resolveServerID(event)
	if err != nil {
		return err
	}

	return p.handler.OnDelete(ctx, id)
}

func (p *messageProcessor) onUpdate(ctx context.Context, event debeziumMessage) error {

	if event.After == nil {
		return nil
	}

	if event.After.DeletedAt != nil {
		return p.handler.OnDelete(ctx, event.After.ID)
	}

	return p.handler.OnUpdate(ctx, event.After.ID, event.After.CreatedByID)
}

func (p *messageProcessor) onCreate(ctx context.Context, event debeziumMessage) error {

	if event.After == nil {
		return nil
	}

	return p.handler.OnCreate(ctx, event.After.ID, event.After.CreatedByID)
}

func (p *messageProcessor) handleMessage(ctx context.Context, event debeziumMessage) error {

	switch event.Op {
	case "c", "r":
		return p.onCreate(ctx, event)

	case "u":
		return p.onUpdate(ctx, event)

	case "d":
		return p.onDelete(ctx, event)

	default:
		return fmt.Errorf("unexpected operation: %s", event.Op)
	}
}

func (p *messageProcessor) ProcessMessage(ctx context.Context, msg redis.XMessage) (canAck bool) {

	event, err := unmarshalDebeziumMessage(msg)
	if err != nil {
		p.logger.Warn("unmarshal message", slog.String("id", msg.ID), slog.Any("error", err))
		return false
	}

	serverID, err := resolveServerID(event)
	if err != nil {
		p.logger.Warn("resolve server id", slog.String("id", msg.ID), slog.Any("error", err))
		return false
	}

	// ponytail: skip stale, out-of-order reprocessed messages (e.g. reclaim brings an
	// older create back after a newer delete for the same server already applied).
	// Comparing the ms-seq stream ids is enough; per-server offset is redis-persisted.
	stale, err := p.isStale(ctx, serverID, msg.ID)
	if err != nil {
		p.logger.Warn("check offset", slog.String("id", msg.ID), slog.Any("error", err))
		return false
	}

	if stale {
		return true
	}

	if err := p.handleMessage(ctx, event); err != nil {
		p.logger.Error("handle server",
			slog.Uint64("server_id", uint64(serverID)),
			slog.String("op", event.Op),
			slog.Any("error", err),
		)
		return false
	}

	if err := p.offsets.SetOffset(ctx, serverID, msg.ID); err != nil {
		p.logger.Error("set offset", slog.String("id", msg.ID), slog.Any("error", err))
	}

	return true
}

// isStale reports whether a message for serverID is older than one already applied.
func (p *messageProcessor) isStale(ctx context.Context, serverID uint, msgID string) (bool, error) {

	applied, err := p.offsets.GetOffset(ctx, serverID)
	if err == redis.Nil {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return p.offsets.IsNewer(applied, msgID)
}
