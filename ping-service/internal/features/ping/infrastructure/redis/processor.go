package redis

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type debeziumMessage struct {
	Before *debeziumEndpointData `json:"before"`
	After  *debeziumEndpointData `json:"after"`
	Op     string                `json:"op"` // c=create, u=update, d=delete
}

type debeziumEndpointData struct {
	ID           uint   `json:"id"`
	ServerID     uint   `json:"server_id"`
	URL          string `json:"url"`
	Method       string `json:"method"`
	ExpectedCode int    `json:"expected_code"`
	Interval     int64  `json:"interval"`
	Timeout      int64  `json:"timeout"`
}

func (d *debeziumEndpointData) toDomain() domain.Endpoint {
	return domain.Endpoint{
		Model:        gorm.Model{ID: d.ID},
		ServerID:     d.ServerID,
		URL:          d.URL,
		Method:       d.Method,
		ExpectedCode: d.ExpectedCode,
		Interval:     time.Duration(d.Interval),
		Timeout:      time.Duration(d.Timeout),
	}
}

type messageProcessor struct {
	handler EndpointEventHandler
	logger  *slog.Logger
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

	domain := event.After.toDomain()
	if err := p.handler.OnUpdate(ctx, domain); err != nil {
		return err
	}

	return nil
}

func (p *messageProcessor) onCreate(ctx context.Context, event debeziumMessage) error {

	if event.After == nil {
		return nil
	}

	domain := event.After.toDomain()
	if err := p.handler.OnCreate(ctx, domain); err != nil {
		return err
	}

	return nil
}

func (p *messageProcessor) ProcessMessage(ctx context.Context, msg redis.XMessage) (canAck bool) {

	raw, ok := msg.Values["value"]
	if !ok {
		p.logger.Warn("stream message missing value field", slog.String("id", msg.ID))
		return false
	}

	rawStr, ok := raw.(string)
	if !ok {
		p.logger.Warn(
			"stream message value not string",
			slog.String("id", msg.ID),
		)

		return false
	}

	event := debeziumMessage{}
	if err := json.Unmarshal([]byte(rawStr), &event); err != nil {

		p.logger.Error(
			"stream message invalid json",
			slog.String("id", msg.ID),
			slog.Any("error", err),
		)

		return false
	}

	switch event.Op {
	case "c":
		if err := p.onCreate(ctx, event); err != nil {
			p.logger.Error("handle endpoint",
				slog.Uint64("endpoint_id", uint64(event.After.ID)),
				slog.String("op", event.Op),
				slog.Any("error", err),
			)

			return false
		}

	case "u":
		if err := p.onUpdate(ctx, event); err != nil {
			p.logger.Error("handle endpoint",
				slog.Uint64("endpoint_id", uint64(event.After.ID)),
				slog.String("op", event.Op),
				slog.Any("error", err),
			)

			return false
		}

	case "d":
		if err := p.onDelete(ctx, event); err != nil {
			p.logger.Error("handle endpoint",
				slog.Uint64("endpoint_id", uint64(event.Before.ID)),
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
