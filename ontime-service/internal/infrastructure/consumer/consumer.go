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
	dlqStreamKey    = streamKey + ".dlq"
	consumerGroup   = "ontime-service-owners"
	consumerName    = "worker-1"
	streamReadCount = 10
	streamBlockTime = 5 * time.Second
)

// reclaimIdleTime is the pending-message age after which a dead worker's
// unacked messages are reclaimed by a live consumer. Exposed as a var so
// tests can lower it.
var reclaimIdleTime = time.Minute

type ServerOwnerHandler interface {
	OnCreate(ctx context.Context, serverID, userID uint) error
	OnUpdate(ctx context.Context, serverID, userID uint) error
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
		client:  c.client,
		offsets: NewRedisOffsetStore(c.client, time.Minute),
	}

	readArgs := redis.XReadGroupArgs{
		Group:    consumerGroup,
		Consumer: consumerName,
		Streams:  []string{streamKey, ">"},
		Count:    streamReadCount,
		Block:    streamBlockTime,
	}

	for ctx.Err() == nil {

		streams, err := c.client.XReadGroup(ctx, &readArgs).Result()

		if err != nil && err != redis.Nil {
			c.logger.Error("stream read", slog.Any("error", err))
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				if processor.ProcessMessage(ctx, msg) {
					c.ack(ctx, msg.ID)
				}
			}
		}

		claimed, err := c.reclaim(ctx)
		if err != nil {
			c.logger.Error("claim pending", slog.Any("error", err))
			continue
		}

		for _, msg := range claimed {
			if processor.ProcessMessage(ctx, msg) {
				c.ack(ctx, msg.ID)
			}
		}
	}
}

// reclaim redis overlapping messages left unacked > reclaimIdleTime (e.g. dead workers) back to this consumer.
func (c *OwnershipConsumer) reclaim(ctx context.Context) ([]redis.XMessage, error) {

	msgID := "0"
	claimed := []redis.XMessage{}

	args := redis.XAutoClaimArgs{
		Stream:   streamKey,
		Group:    consumerGroup,
		Consumer: consumerName,
		MinIdle:  reclaimIdleTime,
		Start:    msgID,
		Count:    streamReadCount,
	}

	for {

		messages, next, err := c.client.XAutoClaim(ctx, &args).Result()
		if err != nil {
			return nil, err
		}

		claimed = append(claimed, messages...)

		if len(messages) == 0 || next == msgID {
			break
		}
		msgID = next

		if len(messages) < streamReadCount {
			break
		}
	}

	return claimed, nil
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
