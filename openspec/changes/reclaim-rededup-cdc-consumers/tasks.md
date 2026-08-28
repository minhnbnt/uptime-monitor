## 1. ontime-service: RedisOffsetStore

- [ ] 1.1 Tạo `ontime-service/internal/infrastructure/consumer/idempotency.go` với `RedisOffsetStore` (key `offset:ontime:serverID:%d`, staleDuration `time.Minute`): `NewRedisOffsetStore`, `GetOffset`, `SetOffset`, `parseOffset`, `IsNewer`. Verify: build xanh.

## 2. ontime-service: processor rededup

- [ ] 2.1 `processor.go`: thêm field `offsets *RedisOffsetStore`; hàm `resolveServerID(event)` (`after.ID` ?? `before.ID`, thiếu → `permanent`); hàm `isStale(ctx, serverID, msgID)` (redis.Nil → không stale; lỗi → không stale; else `IsNewer`).
- [ ] 2.2 Tích hợp vào `ProcessMessage`: unmarshal (lỗi→DLQ) → `resolveServerID` (lỗi→DLQ permanent) → `isStale` (lỗi→retry `false`; stale→ack `true`) → switch op (giữ DLQ) → thành công → `SetOffset`. Verify: `go build ./...` và `go vet ./...` xanh.

## 3. ontime-service: reclaim pending

- [ ] 3.1 `consumer.go`: thêm `var reclaimIdleTime = time.Minute`; hàm `reclaim(ctx)` dùng `XAutoClaim` (loop qua `next`); construct `offsets: NewRedisOffsetStore(c.client, time.Minute)` trong `Run`; gọi `reclaim` sau mỗi batch và xử lý + ack các message reclaim được. Verify: `golangci-lint run --fix ./internal/infrastructure/consumer/...` = 0 issue.

## 4. ping-service: RedisOffsetStore + processor + reclaim

- [ ] 4.1 Tạo `ping-service/internal/infrastructure/redis/idempotency.go` (`RedisOffsetStore` key `offset:ping:endpointID:%d`, staleDuration `30*time.Minute`): `NewRedisOffsetStore`, `GetOffset`, `SetOffset`, `parseOffset`, `IsNewer`.
- [ ] 4.2 `processor.go`: field `offsets`; `resolveEndpointID` (thiếu id → `permanent`); `isStale`; tích hợp vào `ProcessMessage` (giữ DLQ, no-id→DLQ permanent, stale→ack true, thành công→`SetOffset`).
- [ ] 4.3 `redisconsumer.go`: `var reclaimIdleTime = time.Minute`; hàm `reclaim(ctx)` `XAutoClaim`; construct `offsets: NewRedisOffsetStore(c.client, 30*time.Minute)` trong `Run`; gọi `reclaim` sau batch. Verify: `go build`/`go vet`/`golangci-lint run --fix ./internal/infrastructure/redis/...` xanh.

## 5. Tests (testcontainer)

- [ ] 5.1 ontime: thêm `TestMain` + `testRedisAddr` (dùng `internal/infrastructure/testcontainers/redis.go`); test `IsNewer`/`parseOffset` (pure); test `isStale` (set offset → message cũ hơn stale/ack không gọi handler, mới hơn → áp + set offset); test integration `reclaim` (tạo pending qua consumer khác, chạy worker, assert reclaim + xử lý + ack, override `reclaimIdleTime` ~100ms).
- [ ] 5.2 ping: mở rộng `processor_test.go` (đã có `TestMain`); test `isStale` tương tự; test integration `reclaim`. Verify: `go test ./internal/infrastructure/consumer/...` và `./internal/infrastructure/redis/...` xanh (cần Docker).

## 6. Verify & commit

- [ ] 6.1 `golangci-lint run --fix ./...` cho cả ontime-service và ping-service = 0 issue.
- [ ] 6.2 `go test ./...` hai service xanh. Commit riêng ontime và ping (chưa push) theo quy ước repo.
