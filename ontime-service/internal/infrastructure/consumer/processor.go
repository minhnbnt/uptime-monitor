package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type debeziumServerData struct {
	ID          uint       `json:"id"`
	CreatedByID uuid.UUID  `json:"created_by_id"`
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
	offsets *RedisOffsetStore
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

	serverID, err := resolveServerID(event)
	if err != nil {
		p.logger.Warn("stream message no server id", slog.String("id", msg.ID), slog.String("op", event.Op))
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

	switch event.Op {
	case "c", "r":
		err = p.onCreate(ctx, event)
	case "u":
		err = p.onUpdate(ctx, event)
	case "d":
		err = p.onDelete(ctx, event)
	default:
		p.logger.Warn("unknown operation", slog.String("op", event.Op))
		return true
	}

	if err != nil {
		p.logger.Error("handle server",
			slog.Uint64("server_id", uint64(serverID)),
			slog.String("op", event.Op),
			slog.Any("error", err),
		)
		return false
	}

	// record the last applied id so stale reclaims for this server get skipped
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

func resolveServerID(event debeziumMessage) (uint, error) {

	if event.After != nil {
		return event.After.ID, nil
	}

	if event.Before != nil {
		return event.Before.ID, nil
	}

	return 0, errors.New("resolveServerID: event has no before or after")
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
