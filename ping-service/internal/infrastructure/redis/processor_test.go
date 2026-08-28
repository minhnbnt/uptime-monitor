package redis

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/testcontainers"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

var testRedisAddr string

func TestMain(m *testing.M) {
	ctx := context.Background()
	_, addr := testcontainers.StartRedisAddr(ctx)
	testRedisAddr = addr
	os.Exit(m.Run())
}

func TestDebeziumEndpointDataToDomain(t *testing.T) {
	data := debeziumEndpointData{
		ID:           1,
		URL:          "https://example.com",
		Method:       "GET",
		ExpectedCode: 200,
		Interval:     30000000000,
		Timeout:      10000000000,
	}

	got := data.toDomain()
	if got.ID != 1 {
		t.Errorf("ID = %d, want 1", got.ID)
	}
	if got.URL != "https://example.com" {
		t.Errorf("URL = %q", got.URL)
	}
	if got.Method != "GET" {
		t.Errorf("Method = %q", got.Method)
	}
	if got.ExpectedCode != 200 {
		t.Errorf("ExpectedCode = %d, want 200", got.ExpectedCode)
	}
	if got.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", got.Interval)
	}
	if got.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", got.Timeout)
	}
}

func TestResolveDeletedID(t *testing.T) {
	t.Run("before field exists", func(t *testing.T) {
		id, err := resolveDeletedID(debeziumMessage{
			Before: &debeziumEndpointData{ID: 1},
			After:  &debeziumEndpointData{ID: 2},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != 1 {
			t.Errorf("got %d, want 1", id)
		}
	})

	t.Run("only after field exists", func(t *testing.T) {
		id, err := resolveDeletedID(debeziumMessage{
			After: &debeziumEndpointData{ID: 3},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != 3 {
			t.Errorf("got %d, want 3", id)
		}
	})

	t.Run("neither field exists", func(t *testing.T) {
		_, err := resolveDeletedID(debeziumMessage{})
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrPermanent) {
			t.Errorf("expected permanent error, got %v", err)
		}
	})
}

func xmessage(id, value string) redis.XMessage {
	return redis.XMessage{
		ID: id,
		Values: map[string]any{
			"value": value,
		},
	}
}

func TestProcessMessage(t *testing.T) {
	ctx := context.Background()

	t.Run("missing value field is dead-lettered", func(t *testing.T) {
		testcontainers.SkipIfShort(t)
		client := testcontainers.NewTestRedis(t, testRedisAddr)
		log, capLog := logger.NewCapturingLogger()
		p := &messageProcessor{logger: log, client: client}

		msg := redis.XMessage{ID: "1-0", Values: map[string]any{}}
		if !p.ProcessMessage(ctx, msg) {
			t.Error("expected canAck=true for permanent error")
		}
		if !capLog.HasError() {
			t.Error("expected error log (dead-lettered)")
		}
		assertDLQCount(t, client, 1)
	})

	t.Run("value not a string is dead-lettered", func(t *testing.T) {
		testcontainers.SkipIfShort(t)
		client := testcontainers.NewTestRedis(t, testRedisAddr)
		log, capLog := logger.NewCapturingLogger()
		p := &messageProcessor{logger: log, client: client}

		msg := redis.XMessage{ID: "1-0", Values: map[string]any{"value": 42}}
		if !p.ProcessMessage(ctx, msg) {
			t.Error("expected canAck=true for permanent error")
		}
		if !capLog.HasError() {
			t.Error("expected error log (dead-lettered)")
		}
		assertDLQCount(t, client, 1)
	})

	t.Run("invalid JSON is dead-lettered", func(t *testing.T) {
		testcontainers.SkipIfShort(t)
		client := testcontainers.NewTestRedis(t, testRedisAddr)
		log, capLog := logger.NewCapturingLogger()
		p := &messageProcessor{logger: log, client: client}

		canAck := p.ProcessMessage(ctx, xmessage("1-0", "not json"))
		if !canAck {
			t.Error("expected canAck=true for permanent error")
		}
		if !capLog.HasError() {
			t.Error("expected error log (dead-lettered)")
		}
		assertDLQCount(t, client, 1)
	})

	t.Run("unknown operation is dead-lettered", func(t *testing.T) {
		testcontainers.SkipIfShort(t)
		client := testcontainers.NewTestRedis(t, testRedisAddr)
		log, capLog := logger.NewCapturingLogger()
		p := &messageProcessor{logger: log, client: client}

		canAck := p.ProcessMessage(ctx, xmessage("1-0", `{"op":"x"}`))
		if !canAck {
			t.Error("expected canAck=true for permanent error")
		}
		if !capLog.HasError() {
			t.Error("expected error log (dead-lettered)")
		}
		assertDLQCount(t, client, 1)
	})

	t.Run("create operation", func(t *testing.T) {
		var created domain.Endpoint
		p := &messageProcessor{
			handler: &mockEndpointEventHandler{
				onCreateFn: func(_ context.Context, inEp domain.Endpoint) error {
					created = inEp
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		canAck := p.ProcessMessage(ctx, xmessage("1-0", `{"op":"c","after":{"id":1,"server_id":10,"url":"https://example.com","method":"GET","expected_code":200,"interval":30000000000,"timeout":10000000000}}`))
		if !canAck {
			t.Error("expected canAck=true")
		}
		if created.ID != 1 {
			t.Errorf("created endpoint ID = %d, want 1", created.ID)
		}
	})

	t.Run("create operation handler error", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		p := &messageProcessor{
			handler: &mockEndpointEventHandler{
				onCreateFn: func(_ context.Context, _ domain.Endpoint) error {
					return errors.New("handler error")
				},
			},
			logger: log,
		}

		canAck := p.ProcessMessage(ctx, xmessage("1-0", `{"op":"c","after":{"id":1}}`))
		if canAck {
			t.Error("expected canAck=false")
		}
		if !capLog.HasError() {
			t.Error("expected error log")
		}
	})

	t.Run("update operation", func(t *testing.T) {
		var updated domain.Endpoint
		p := &messageProcessor{
			handler: &mockEndpointEventHandler{
				onUpdateFn: func(_ context.Context, inEp domain.Endpoint) error {
					updated = inEp
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		canAck := p.ProcessMessage(ctx, xmessage("1-0", `{"op":"u","after":{"id":2,"url":"https://updated.com"}}`))
		if !canAck {
			t.Error("expected canAck=true")
		}
		if updated.ID != 2 {
			t.Errorf("updated endpoint ID = %d, want 2", updated.ID)
		}
	})

	t.Run("delete operation", func(t *testing.T) {
		var deletedID uint
		p := &messageProcessor{
			handler: &mockEndpointEventHandler{
				onDeleteFn: func(_ context.Context, id uint) error {
					deletedID = id
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		canAck := p.ProcessMessage(ctx, xmessage("1-0", `{"op":"d","before":{"id":3}}`))
		if !canAck {
			t.Error("expected canAck=true")
		}
		if deletedID != 3 {
			t.Errorf("deleted ID = %d, want 3", deletedID)
		}
	})

	t.Run("create with nil after acks but does not call handler", func(t *testing.T) {
		var handlerCalled bool
		p := &messageProcessor{
			handler: &mockEndpointEventHandler{
				onCreateFn: func(_ context.Context, _ domain.Endpoint) error {
					handlerCalled = true
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		canAck := p.ProcessMessage(ctx, xmessage("1-0", `{"op":"c"}`))
		if handlerCalled {
			t.Error("OnCreate should not be called after is nil")
		}
		if !canAck {
			t.Error("expected canAck=true — tombstone events should be acked")
		}
	})

	t.Run("delete on event with only after resolves ID from after", func(t *testing.T) {
		var deletedID uint
		p := &messageProcessor{
			handler: &mockEndpointEventHandler{
				onDeleteFn: func(_ context.Context, id uint) error {
					deletedID = id
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		canAck := p.ProcessMessage(ctx, xmessage("1-0", `{"op":"d","after":{"id":5}}`))
		if !canAck {
			t.Error("expected canAck=true")
		}
		if deletedID != 5 {
			t.Errorf("deleted ID = %d, want 5", deletedID)
		}
	})

	t.Run("update with nil after acks but does not call handler", func(t *testing.T) {
		var handlerCalled bool
		p := &messageProcessor{
			handler: &mockEndpointEventHandler{
				onUpdateFn: func(_ context.Context, _ domain.Endpoint) error {
					handlerCalled = true
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		canAck := p.ProcessMessage(ctx, xmessage("1-0", `{"op":"u"}`))
		if handlerCalled {
			t.Error("OnUpdate should not be called when after is nil")
		}
		if !canAck {
			t.Error("expected canAck=true — tombstone events should be acked")
		}
	})
}

func TestDeadLetterWritesToDLQStream(t *testing.T) {
	testcontainers.SkipIfShort(t)
	ctx := context.Background()
	client := testcontainers.NewTestRedis(t, testRedisAddr)

	p := &messageProcessor{logger: logger.NewMockLogger(), client: client}

	msg := xmessage("1-0", "not json")
	if !p.ProcessMessage(ctx, msg) {
		t.Fatal("expected canAck=true for permanent error")
	}

	entries, err := client.XRange(ctx, dlqStreamKey, "-", "+").Result()
	if err != nil {
		t.Fatalf("xrange dlq: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("dlq entries = %d, want 1", len(entries))
	}
	if entries[0].Values["original_id"] != "1-0" {
		t.Errorf("original_id = %v, want 1-0", entries[0].Values["original_id"])
	}
	if _, ok := entries[0].Values["error"]; !ok {
		t.Error("dlq entry missing error field")
	}
}

func assertDLQCount(t *testing.T, client *redis.Client, want int64) {
	t.Helper()
	ctx := context.Background()
	n, err := client.XLen(ctx, dlqStreamKey).Result()
	if err != nil {
		t.Fatalf("xlen dlq: %v", err)
	}
	if n != want {
		t.Errorf("dlq entries = %d, want %d", n, want)
	}
}

func TestOnCreateUpdateDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("onCreate delegates", func(t *testing.T) {
		var got domain.Endpoint
		p := &messageProcessor{
			handler: &mockEndpointEventHandler{
				onCreateFn: func(_ context.Context, inE domain.Endpoint) error {
					got = inE
					return nil
				},
			},
		}
		err := p.onCreate(ctx, debeziumMessage{After: &debeziumEndpointData{ID: 1}})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got.ID != 1 {
			t.Errorf("got ID %d, want 1", got.ID)
		}
	})

	t.Run("onCreate nil after returns nil", func(t *testing.T) {
		var handlerCalled bool
		p := &messageProcessor{
			handler: &mockEndpointEventHandler{
				onCreateFn: func(_ context.Context, _ domain.Endpoint) error {
					handlerCalled = true
					return nil
				},
			},
		}
		err := p.onCreate(ctx, debeziumMessage{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if handlerCalled {
			t.Error("OnCreate should not be called")
		}
	})

	t.Run("onUpdate delegates", func(t *testing.T) {
		var got domain.Endpoint
		p := &messageProcessor{
			handler: &mockEndpointEventHandler{
				onUpdateFn: func(_ context.Context, inE domain.Endpoint) error {
					got = inE
					return nil
				},
			},
		}
		err := p.onUpdate(ctx, debeziumMessage{After: &debeziumEndpointData{ID: 2}})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got.ID != 2 {
			t.Errorf("got ID %d, want 2", got.ID)
		}
	})

	t.Run("onUpdate nil after returns nil", func(t *testing.T) {
		var handlerCalled bool
		p := &messageProcessor{
			handler: &mockEndpointEventHandler{
				onUpdateFn: func(_ context.Context, _ domain.Endpoint) error {
					handlerCalled = true
					return nil
				},
			},
		}
		err := p.onUpdate(ctx, debeziumMessage{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if handlerCalled {
			t.Error("OnUpdate should not be called")
		}
	})

	t.Run("onDelete delegates", func(t *testing.T) {
		var gotID uint
		p := &messageProcessor{
			handler: &mockEndpointEventHandler{
				onDeleteFn: func(_ context.Context, id uint) error {
					gotID = id
					return nil
				},
			},
		}
		err := p.onDelete(ctx, debeziumMessage{Before: &debeziumEndpointData{ID: 3}})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if gotID != 3 {
			t.Errorf("got ID %d, want 3", gotID)
		}
	})
}
