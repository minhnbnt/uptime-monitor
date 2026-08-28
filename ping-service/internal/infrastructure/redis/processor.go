package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"

	"github.com/redis/go-redis/v9"
)

type messageProcessor struct {
	handler EndpointEventHandler
	logger  *slog.Logger
	client  *redis.Client
	offsets *OffsetStore
}

func (p *messageProcessor) onDelete(ctx context.Context, event debeziumMessage) error {

	id, err := resolveDeletedID(event)
	if err != nil {
		return err
	}

	if err := p.handler.OnDelete(ctx, id); err != nil {
		return err
	}

	return nil
}

func (p *messageProcessor) onUpdate(ctx context.Context, event debeziumMessage) error {

	if event.After == nil {
		return nil
	}

	endpoint := event.After.toDomain()
	if err := p.handler.OnUpdate(ctx, endpoint); err != nil {
		return err
	}

	return nil
}

func (p *messageProcessor) onCreate(ctx context.Context, event debeziumMessage) error {

	if event.After == nil {
		return nil
	}

	endpoint := event.After.toDomain()
	if err := p.handler.OnCreate(ctx, endpoint); err != nil {
		return err
	}

	return nil
}

func (p *messageProcessor) ProcessMessage(ctx context.Context, msg redis.XMessage) (canAck bool) {

	event, err := unmarshalDebeziumMessage(msg)
	if err != nil {
		return p.deadLetterOrRetry(ctx, msg, err)
	}

	endpointID, err := resolveEndpointID(event)
	if err != nil {
		return p.deadLetterOrRetry(ctx, msg, err)
	}

	// ponytail: skip stale, out-of-order reprocessed messages (e.g. reclaim brings an
	// older create back after a newer delete for the same endpoint already applied).
	// Comparing the ms-seq stream ids is enough; per-endpoint offset is redis-persisted.
	stale, err := p.isStale(ctx, endpointID, msg.ID)
	if err != nil {
		p.logger.Warn("check offset", slog.String("id", msg.ID), slog.Any("error", err))
		return false
	}
	if stale {
		return true
	}

	switch event.Op {
	case "c", "r":
		if err := p.onCreate(ctx, event); err != nil {
			return p.deadLetterOrRetry(ctx, msg, fmt.Errorf("handle endpoint: %w", err))
		}

	case "u":
		if err := p.onUpdate(ctx, event); err != nil {
			return p.deadLetterOrRetry(ctx, msg, fmt.Errorf("handle endpoint: %w", err))
		}

	case "d":
		if err := p.onDelete(ctx, event); err != nil {
			return p.deadLetterOrRetry(ctx, msg, fmt.Errorf("handle endpoint: %w", err))
		}

	default:
		return p.deadLetterOrRetry(ctx, msg, permanent(fmt.Errorf("unexpected operation %q", event.Op)))
	}

	if p.offsets != nil {
		if err := p.offsets.SetOffset(ctx, endpointID, msg.ID); err != nil {
			p.logger.Error("set offset", slog.String("id", msg.ID), slog.Any("error", err))
		}
	}

	return true
}

// isStale reports whether a message for endpointID is older than one already applied.
func (p *messageProcessor) isStale(ctx context.Context, endpointID uint, msgID string) (bool, error) {

	if p.offsets == nil {
		return false, nil
	}

	applied, err := p.offsets.GetOffset(ctx, endpointID)
	if err == redis.Nil {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return p.offsets.IsNewer(applied, msgID)
}

func resolveEndpointID(event debeziumMessage) (uint, error) {

	if event.After != nil {
		return event.After.ID, nil
	}

	if event.Before != nil {
		return event.Before.ID, nil
	}

	return 0, permanent(errors.New("resolveEndpointID: event has no before or after"))
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
