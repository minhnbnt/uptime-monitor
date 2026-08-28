## Why

Consumer CDC của ontime-service (bảng `servers`) và ping-service (bảng `endpoints`) đọc Redis Stream qua consumer group. Hiện tại:

- Khi worker chết/restart giữa lúc đã read nhưng chưa ack, message nằm trong PEL (pending entries list) → không bao giờ được xử lý lại (mất update) vì chưa có reclaim.
- Redis Stream có thể gửi lại message (reset consumer group, redelivery, hoặc reclaim về sau) → cùng một entity có thể nhận lại sự kiện cũ (vd create sau khi đã có delete) theo thứ tự sai, gây trạng thái không nhất quán.

Ý tưởng lấy từ commit `baccd2c8` (nhánh k8s): "reclaim and rededup CDC ownership/event messages". Mục tiêu là consumer khoẻ hơn — nhận lại message pending từ worker chết và bỏ qua (dedupe) message CDC cũ hơn offset đã áp dụng — mà không cherry-pick toàn bộ commit (nhánh k8s có cấu trúc `service/events` khác master).

## What Changes

- **Reclaim pending messages**: trong vòng lặp `Run`, sau mỗi batch, gọi `XAutoClaim` để nhận lại message đã deliver nhưng chưa ack quá `reclaimIdleTime` (mặc định 1 phút) từ worker chết, rồi xử lý + ack.
- **Rededup theo entity**: thêm `RedisOffsetStore` lưu last-applied message id (định dạng `ms-seq`) theo server id (ontime) / endpoint id (ping) với TTL (1 phút ontime, 30 phút ping). `ProcessMessage` tính `isStale`: nếu message cũ hơn offset đã lưu → ack `true`, bỏ qua (không gọi handler). Sau xử lý thành công → `SetOffset`.
- **`resolveServerID` / `resolveEndpointID`**: lấy entity id từ `after` (hoặc `before` nếu là tombstone) làm key dedupe.
- **ping-service**: áp dụng tương tự vào `internal/infrastructure/redis/` (vì `internal/service/events/` ở commit k8s không tồn tại trên master). Construct `RedisOffsetStore` inline trong `Run` (không đăng ký qua `do`).

## Capabilities

### New Capabilities
- `cdc-consumer-reliability`: độ tin cậy của CDC consumer — reclaim message pending từ worker chết và bỏ qua (dedupe) message CDC cũ hơn offset đã áp dụng, áp dụng cho cả ontime ownership consumer và ping endpoint consumer.

### Modified Capabilities
- (không có)

## Impact

- ontime: `internal/infrastructure/consumer/consumer.go` (reclaim + construct offsets), `processor.go` (field offsets, `resolveServerID`, `isStale`), `idempotency.go` (mới).
- ping: `internal/infrastructure/redis/redisconsumer.go` (reclaim + construct offsets), `processor.go` (field offsets, `resolveEndpointID`, `isStale`), `idempotency.go` (mới).
- Dùng thêm Redis `SET`/`GET` cho offset key (TTL). Không đổi API gRPC/public, không đổi schema DB.

## Lưu ý quyết định (deviation so với commit gốc)

- **No-id → DLQ permanent**: commit gốc trả `false` (retry) khi event thiếu cả `before`/`after` (không lấy được id). Vì master đã có DLQ, trả retry sẽ tái diễn poison loop → chọn DLQ cho message thực sự hỏng.
- **`reclaimIdleTime` thành `var`** (commit gốc inline `time.Minute`) để test override ~100ms.
