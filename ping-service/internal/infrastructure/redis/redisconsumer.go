package redis

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

const (
	streamKey       = "uptime.public.endpoints"
	consumerGroup   = "ping-service"
	consumerName    = "worker-1"
	streamReadCount = 10
	streamBlockTime = 5 * time.Second
)

type StreamEventConsumer struct {
	client *redis.Client
	logger *slog.Logger
}

func RegisterStreamEventConsumer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*StreamEventConsumer, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return &StreamEventConsumer{
			client: wrapper.GetClient(),
			logger: do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

type EndpointEventHandler interface {
	OnCreate(context.Context, domain.Endpoint) error
	OnUpdate(context.Context, domain.Endpoint) error
	OnDelete(ctx context.Context, id uint) error
}

func (c *StreamEventConsumer) Run(ctx context.Context, handler EndpointEventHandler) {

	c.logger.Info(
		"starting stream consumer",
		slog.String("stream", streamKey),
		slog.String("group", consumerGroup),
	)

	err := c.client.XGroupCreateMkStream(ctx, streamKey, consumerGroup, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		c.logger.Warn("create consumer group", slog.Any("error", err))
	}

	processor := &messageProcessor{
		handler: handler,
		logger:  c.logger,
	}

	for ctx.Err() == nil {

		streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumerName,
			Streams:  []string{streamKey, ">"},
			Count:    streamReadCount,
			Block:    streamBlockTime,
		}).Result()

		if err == redis.Nil {
			continue
		}

		if err != nil {
			c.logger.Error("stream read", slog.Any("error", err))
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {

				canAck := processor.ProcessMessage(ctx, msg)

				if canAck {
					c.ack(ctx, msg.ID)
				}
			}
		}
	}
}

func (c *StreamEventConsumer) ack(ctx context.Context, msgID string) {

	err := c.client.XAck(ctx, streamKey, consumerGroup, msgID).Err()

	if err != nil {
		c.logger.Error("ack message",
			slog.String("msg_id", msgID),
			slog.Any("error", err),
		)
	}
}
