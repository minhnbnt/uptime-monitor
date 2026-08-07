package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"

	"github.com/redis/go-redis/v9"
)

type messageProcessor struct {
	handler ServerOwnerHandler
	logger  *slog.Logger
	offsets *RedisOffsetStore
	client  *redis.Client
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
		return permanent(fmt.Errorf("unexpected operation %q", event.Op))
	}
}

func (p *messageProcessor) ProcessMessage(ctx context.Context, msg redis.XMessage) (canAck bool) {

	event, err := unmarshalDebeziumMessage(msg)
	if err != nil {
		return p.deadLetterOrRetry(ctx, msg, err)
	}

	serverID, err := resolveServerID(event)
	if err != nil {
		return p.deadLetterOrRetry(ctx, msg, err)
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
		return p.deadLetterOrRetry(ctx, msg, err)
	}

	if err := p.offsets.SetOffset(ctx, serverID, msg.ID); err != nil {
		p.logger.Error("set offset", slog.String("id", msg.ID), slog.Any("error", err))
	}

	return true
}

// deadLetterOrRetry acks permanent failures into the DLQ stream; transient errors
// are returned for retry via reclaim. A failed DLQ write is NOT acked so the
// message is never lost.
func (p *messageProcessor) deadLetterOrRetry(ctx context.Context, msg redis.XMessage, err error) bool {

	if !errors.Is(err, ErrPermanent) {
		p.logger.Error("handle message",
			slog.String("id", msg.ID),
			slog.Any("error", err),
		)
		return false
	}

	if dlqErr := p.sendToDLQ(ctx, msg, err); dlqErr != nil {
		p.logger.Error("write dlq",
			slog.String("id", msg.ID),
			slog.Any("error", dlqErr),
		)
		return false
	}

	p.logger.Error(
		"dead-lettered",
		slog.String("id", msg.ID),
		slog.Any("error", err),
	)

	return true
}

func (p *messageProcessor) sendToDLQ(ctx context.Context, msg redis.XMessage, err error) error {

	values := maps.Clone(msg.Values)

	values["error"] = err.Error()
	values["original_id"] = msg.ID

	args := redis.XAddArgs{
		Stream: dlqStreamKey,
		Values: values,
	}

	return p.client.XAdd(ctx, &args).Err()
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
