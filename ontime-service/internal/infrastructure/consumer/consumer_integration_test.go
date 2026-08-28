package consumer

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/testcontainers"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/logger"
)

var testRedisAddr string

func TestMain(m *testing.M) {
	ctx := context.Background()
	_, addr := testcontainers.StartRedisAddr(ctx)
	testRedisAddr = addr
	os.Exit(m.Run())
}

func xmessage(id, value string) redis.XMessage {
	return redis.XMessage{
		ID: id,
		Values: map[string]any{
			"value": value,
		},
	}
}

type mockOwnerHandler struct {
	onCreateFn func(context.Context, uint, uint) error
	onUpdateFn func(context.Context, uint, uint) error
	onDeleteFn func(context.Context, uint) error
}

func (m *mockOwnerHandler) OnCreate(ctx context.Context, serverID, userID uint) error {
	if m.onCreateFn == nil {
		return nil
	}
	return m.onCreateFn(ctx, serverID, userID)
}

func (m *mockOwnerHandler) OnUpdate(ctx context.Context, serverID, userID uint) error {
	if m.onUpdateFn == nil {
		return nil
	}
	return m.onUpdateFn(ctx, serverID, userID)
}

func (m *mockOwnerHandler) OnDelete(ctx context.Context, serverID uint) error {
	if m.onDeleteFn == nil {
		return nil
	}
	return m.onDeleteFn(ctx, serverID)
}

var _ ServerOwnerHandler = (*mockOwnerHandler)(nil)

func TestProcessMessageSkipsStaleAndAppliesFresh(t *testing.T) {
	testcontainers.SkipIfShort(t)
	ctx := context.Background()
	client := testcontainers.NewTestRedis(t, testRedisAddr)

	store := NewRedisOffsetStore(client, time.Minute)
	if err := store.SetOffset(ctx, 1, "2-0"); err != nil {
		t.Fatalf("set offset: %v", err)
	}

	var mu sync.Mutex
	var createdServerID uint
	h := &mockOwnerHandler{
		onCreateFn: func(_ context.Context, serverID, _ uint) error {
			mu.Lock()
			createdServerID = serverID
			mu.Unlock()
			return nil
		},
	}
	p := &messageProcessor{
		handler: h,
		logger:  logger.NewMockLogger(),
		client:  client,
		offsets: store,
	}

	// stale message (id 1-0 < applied 2-0) -> acked, handler NOT called
	if !p.ProcessMessage(ctx, xmessage("1-0", `{"op":"c","after":{"id":1,"created_by_id":10}}`)) {
		t.Error("expected canAck=true for stale message")
	}
	mu.Lock()
	if createdServerID != 0 {
		t.Errorf("stale message must not call handler, got serverID %d", createdServerID)
	}
	mu.Unlock()

	// fresh message (id 3-0 > applied 2-0) -> acked, handler called
	if !p.ProcessMessage(ctx, xmessage("3-0", `{"op":"c","after":{"id":1,"created_by_id":10}}`)) {
		t.Error("expected canAck=true for fresh message")
	}
	mu.Lock()
	if createdServerID != 1 {
		t.Errorf("fresh message must call handler with serverID 1, got %d", createdServerID)
	}
	mu.Unlock()
}

func TestOwnershipConsumerRunReclaimsPending(t *testing.T) {
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
	h := &mockOwnerHandler{
		onCreateFn: func(context.Context, uint, uint) error {
			mu.Lock()
			calls++
			mu.Unlock()
			return nil
		},
	}
	consumer := &OwnershipConsumer{client: client, logger: logger.NewMockLogger()}

	// publish a valid server message, then claim it with a different consumer WITHOUT
	// acking it (simulating a dead worker) so it becomes pending.
	if err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]any{"value": `{"op":"c","after":{"id":1,"created_by_id":10}}`},
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
