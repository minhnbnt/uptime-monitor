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
	client  *redis.Client
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

func (p *messageProcessor) ProcessMessage(ctx context.Context, msg redis.XMessage) (canAck bool) {

	event, err := unmarshalDebeziumMessage(msg)
	if err != nil {
		return p.deadLetterOrRetry(ctx, msg, err)
	}

	switch event.Op {
	case "c", "r":
		if err := p.onCreate(ctx, event); err != nil {
			return p.deadLetterOrRetry(ctx, msg, fmt.Errorf("handle server: %w", err))
		}

	case "u":
		if err := p.onUpdate(ctx, event); err != nil {
			return p.deadLetterOrRetry(ctx, msg, fmt.Errorf("handle server: %w", err))
		}

	case "d":
		if err := p.onDelete(ctx, event); err != nil {
			return p.deadLetterOrRetry(ctx, msg, fmt.Errorf("handle server: %w", err))
		}

	default:
		return p.deadLetterOrRetry(ctx, msg, permanent(fmt.Errorf("unexpected operation %q", event.Op)))
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

func resolveDeletedID(event debeziumMessage) (uint, error) {
	if event.Before != nil {
		return event.Before.ID, nil
	}
	if event.After != nil {
		return event.After.ID, nil
	}
	return 0, permanent(errors.New("resolveDeletedID: event has no before or after"))
}
