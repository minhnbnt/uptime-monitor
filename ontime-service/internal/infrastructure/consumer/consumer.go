package consumer

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
)

const (
	streamKey       = "uptime.public.servers"
	consumerGroup   = "ontime-service-owners"
	consumerName    = "worker-1"
	streamReadCount = 10
	streamBlockTime = 5 * time.Second
)

type ServerOwnerHandler interface {
	OnCreate(ctx context.Context, serverID, userID uint) error
	OnUpdate(ctx context.Context, serverID, userID uint, deletedAt *time.Time) error
	OnDelete(ctx context.Context, serverID uint) error
}

type OwnershipConsumer struct {
	client *redis.Client
	logger *slog.Logger
}

func RegisterOwnershipConsumer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OwnershipConsumer, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return &OwnershipConsumer{
			client: wrapper.GetClient(),
			logger: do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (c *OwnershipConsumer) Run(ctx context.Context, handler ServerOwnerHandler) {

	c.logger.Info(
		"starting ownership consumer",
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

func (c *OwnershipConsumer) ack(ctx context.Context, msgID string) {
	err := c.client.XAck(ctx, streamKey, consumerGroup, msgID).Err()
	if err != nil {
		c.logger.Error("ack message",
			slog.String("msg_id", msgID),
			slog.Any("error", err),
		)
	}
}
