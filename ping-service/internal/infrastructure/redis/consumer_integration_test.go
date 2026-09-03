package redis

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/testcontainers"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

func TestStreamEventConsumerRunReclaimsPending(t *testing.T) {
	testcontainers.SkipIfShort(t)

	defer func(old time.Duration) { reclaimIdleTime = old }(reclaimIdleTime)
	reclaimIdleTime = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := testcontainers.NewTestRedis(t, testRedisAddr)
	if err := client.XGroupCreateMkStream(ctx, streamKey, consumerGroup, "$").Err(); err != nil {
		t.Fatalf("create group: %v", err)
	}

	var mu sync.Mutex
	var calls int
	h := &mockEndpointEventHandler{
		onCreateFn: func(context.Context, domain.Endpoint) error {
			mu.Lock()
			calls++
			mu.Unlock()
			return nil
		},
	}
	consumer := &StreamEventConsumer{client: client, logger: logger.NewMockLogger()}

	// publish a valid endpoint message, then claim it with a different consumer WITHOUT
	// acking it (simulating a dead worker) so it becomes pending.
	if err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]any{"value": `{"payload":{"op":"c","after":{"id":1,"url":"https://x.com","method":"GET","expected_code":200,"interval":5000000000,"timeout":2000000000}}}`},
	}).Err(); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    consumerGroup,
		Consumer: "probe",
		Streams:  []string{streamKey, ">"},
		Count:    streamReadCount,
	}).Result(); err != nil && err != redis.Nil {
		t.Fatalf("probe read: %v", err)
	}

	// let the pending entry age past reclaimIdleTime
	time.Sleep(200 * time.Millisecond)

	go consumer.Run(ctx, h)

	deadline := time.Now().Add(15 * time.Second)
	for {
		mu.Lock()
		c := calls
		mu.Unlock()
		if c >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout: handler calls = %d", c)
		}
		time.Sleep(50 * time.Millisecond)
	}

	pending, err := client.XPending(ctx, streamKey, consumerGroup).Result()
	if err != nil {
		t.Fatalf("xpending: %v", err)
	}
	if pending.Count != 0 {
		t.Errorf("expected no pending entries after reclaim+ack, got %d", pending.Count)
	}
}
