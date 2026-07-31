## 1. Debezium config

- [ ] 1.1 `config/debezium.yml` — `table.include.list` → `public.servers,public.server_http_configs`

## 2. CDC processor (RawMessage + switch stream)

- [ ] 2.1 `processor.go` — thay `debeziumMessage` bằng `cdcMessage{Before, After json.RawMessage, Op string}`; `messageProcessor.ProcessMessage(ctx, stream string, msg)` — parse `value` → unmarshal `cdcMessage` → `switch stream` (`ServerStreamKey`/`HTTPConfigStreamKey`, default → warn + false)
- [ ] 2.2 `processor.go` — `handleServer(ctx, *cdcMessage) error`: dispatch op (`c/r`→OnCreate, `u`→OnUpdate qua `debeziumServerData.toDomain()`, `d`→OnDelete), nil-After → ack không gọi handler
- [ ] 2.3 `processor.go` — helper `deletedData(*cdcMessage) (json.RawMessage, error)` (trả `Before` nếu có, fallback `After`; lỗi nếu cả 2 nil) — dùng chung cho op `d` của cả 2 bảng
- [ ] 2.4 `processor.go` — `messageProcessor` thêm field `httpHandler HTTPConfigEventHandler` + constructor `NewMessageProcessor(handler ServerEventHandler, httpHandler HTTPConfigEventHandler, logger)`

## 3. HTTP config handling (mới)

- [ ] 3.1 `httpconfig_processor.go` — `debeziumHTTPConfigData` (ServerID/Port/EndpointPath/ExpectedCode/BodyCheckExpr/Method, json tags khớp cột DB) + `toDomain() *domain.ServerHTTPConfig`
- [ ] 3.2 `httpconfig_processor.go` — `HTTPConfigEventHandler` interface (`OnSet(ctx, id, *ServerHTTPConfig)`, `OnClear(ctx, id)`)
- [ ] 3.3 `httpconfig_processor.go` — `handleHTTPConfig(ctx, *cdcMessage) error`: `c/r/u`→OnSet (unmarshal After, nil-After → ack), `d`→OnClear (unmarshal `deletedData` → ServerID)

## 4. Multi-stream consumer

- [ ] 4.1 `redisconsumer.go` — export consts `ServerStreamKey = "uptime.public.servers"`, `HTTPConfigStreamKey = "uptime.public.server_http_configs"`; `StreamProcessor` interface `ProcessMessage(ctx, stream string, msg) bool`
- [ ] 4.2 `redisconsumer.go` — `Run(ctx, streams []string, p StreamProcessor)` — tạo group cho từng stream, `XReadGroup` đọc tất cả, dispatch theo `stream.Stream`, `XAck` đúng stream

## 5. Cache write paths

- [ ] 5.1 `endpointcache.go` — extract `httpConfigHashValues(cfg)`; `SetMulti` dùng chung helper
- [ ] 5.2 `endpointcache.go` — `SetHTTPConfig(ctx, id, cfg)` (HSet + refresh TTL)
- [ ] 5.3 `endpointcache.go` — `ClearHTTPConfig(ctx, id)` (HDel marker + 5 field)

## 6. Service wiring

- [ ] 6.1 `endpointevent.go` — `HTTPConfigEventHandler` struct (wrap `ServerMetaCache`: OnSet→SetHTTPConfig, OnClear→ClearHTTPConfig)
- [ ] 6.2 `endpointevent.go` — `EndpointEventService` thêm httpHandler + logger; `Run(ctx)` gọi `s.consumer.Run(ctx, []string{redis.ServerStreamKey, redis.HTTPConfigStreamKey}, redis.NewMessageProcessor(...))`

## 7. Tests & verify

- [ ] 7.1 `processor_test.go` — đổi sang `NewMessageProcessor(...)`; `ProcessMessage(ctx, stream, msg)` truyền stream key; test `deletedData`; bỏ `TestOnCreateUpdateDelete` (đã phủ bởi `TestProcessMessage`)
- [ ] 7.2 `mocks_test.go` — thêm `mockHTTPConfigEventHandler`
- [ ] 7.3 `httpconfig_processor_test.go` (mới) — qua stream `HTTPConfigStreamKey`: OnSet (`c`/`u`), OnClear (`d`), nil-After ack, invalid json/value
- [ ] 7.4 `scheduler_integration_test.go` — round-trip `SetHTTPConfig`/`ClearHTTPConfig` → `MGet` (SkipIfShort)
- [ ] 7.5 `go build ./...`, `go vet ./...`, `golangci-lint run ./...`, `go test ./internal/infrastructure/redis/ ./internal/infrastructure/scheduler/ ./internal/service/` xanh
