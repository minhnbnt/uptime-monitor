package redis

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type debeziumMessage struct {
	Before *debeziumServerData `json:"before"`
	After  *debeziumServerData `json:"after"`
	Op     string              `json:"op"` // c=create, u=update, d=delete
}

type debeziumServerData struct {
	ID            uint    `json:"id"`
	Namespace     string  `json:"namespace"`
	Kind          string  `json:"kind"`
	ObjectID      string  `json:"object_id"`
	ContainerName string  `json:"container_name"`
	Interval      int64   `json:"interval"`
	Timeout       int64   `json:"timeout"`
	PingType      uint    `json:"ping_type"`
	Port          int     `json:"port"`
	EndpointPath  string  `json:"endpoint_path"`
	ExpectedCode  int     `json:"expected_code"`
	BodyCheckExpr *string `json:"body_check_expr"`
	Method        string  `json:"method"`
}

func (d *debeziumServerData) toDomain() domain.Server {
	return domain.Server{
		ID:            d.ID,
		Namespace:     d.Namespace,
		Kind:          d.Kind,
		ObjectID:      d.ObjectID,
		ContainerName: d.ContainerName,
		Interval:      time.Duration(d.Interval),
		Timeout:       time.Duration(d.Timeout),
		PingType:      d.PingType,
		Port:          d.Port,
		EndpointPath:  d.EndpointPath,
		ExpectedCode:  d.ExpectedCode,
		BodyCheckExpr: d.BodyCheckExpr,
		Method:        d.Method,
	}
}

type messageProcessor struct {
	handler ServerEventHandler
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

	sv := event.After.toDomain()
	if err := p.handler.OnUpdate(ctx, sv); err != nil {
		return err
	}

	return nil
}

func (p *messageProcessor) onCreate(ctx context.Context, event debeziumMessage) error {

	if event.After == nil {
		return nil
	}

	sv := event.After.toDomain()
	if err := p.handler.OnCreate(ctx, sv); err != nil {
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
