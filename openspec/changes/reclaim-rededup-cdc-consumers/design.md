## Context

Master đã có DLQ cho cả hai consumer (commit `b122b8c` ontime, `262a97f` ping): lỗi permanent → ghi `*.dlq` stream + ack; lỗi transient → retry. Nhưng chưa có (1) reclaim message pending từ worker chết, và (2) dedupe message CDC cũ hơn offset đã áp. Nhánh k8s có cả hai trong commit `baccd2c8`, nhưng cấu trúc ping-service của nó khác master (dùng `internal/service/events/`). Xem proposal.md - Why.

## Goals / Non-Goals

**Goals:**
- Nhận lại message pending quá idle qua `XAutoClaim` cho cả ontime và ping.
- Dedupe theo entity bằng `RedisOffsetStore` (last-applied `ms-seq` id, TTL).
- Giữ nguyên hành vi DLQ đã có; tích hợp dedup không phá vỡ DLQ.

**Non-Goals:**
- Không port refactor `internal/service/events/` của nhánh k8s (không tồn tại trên master).
- Không đổi API gRPC/public, không đổi schema DB.
- Không xây dựng khung offset chung qua DI `do` cho ping (construct inline).

## Decisions

1. **Reclaim bằng `XAutoClaim`**: thêm `reclaim(ctx)` trong consumer. `XAutoClaimArgs{Stream, Group, Consumer, MinIdle: reclaimIdleTime, Start: "0", Count: streamReadCount}`; loop qua `next` đến khi hết. Gọi sau mỗi batch trong `Run`. `reclaimIdleTime` là `var` (default `time.Minute`) để test override ~100ms.

2. **`RedisOffsetStore` (idempotency.go mới)**:
   - ontime: key `offset:ontime:serverID:%d`, staleDuration `time.Minute`.
   - ping: key `offset:ping:endpointID:%d`, staleDuration `30 * time.Minute`.
   - API: `GetOffset(ctx, id) (string, error)`, `SetOffset(ctx, id, offset) error`, `parseOffset` (`fmt.Sscanf(offset, "%d-%d", &ms, &seq)`), `IsNewer(a, b)` so sánh `ms` rồi `seq`.
   - Construct inline trong `Run`: `NewRedisOffsetStore(c.client, time.Minute)` (ontime) / `30*time.Minute` (ping). Không đăng ký `do` cho ping (trái với k8s, vì master construct trực tiếp).

3. **Processor tích hợp dedup**:
   - Thêm field `offsets *RedisOffsetStore`.
   - `resolveServerID` / `resolveEndpointID`: `after.ID` nếu có, ngược lại `before.ID`; thiếu cả hai → `permanent` error.
   - `isStale(ctx, id, msgID)`: `GetOffset` → `redis.Nil` → không stale; lỗi khác → không stale (retry an toàn); có offset → `IsNewer(offset, msgID)`.
   - `ProcessMessage` (giữ thứ tự DLQ): unmarshal (lỗi→DLQ) → resolveID (lỗi→DLQ permanent) → isStale (lỗi→retry `false`; stale→ack `true`, bỏ qua) → switch op (handler lỗi→DLQ transient/permanent) → thành công → `SetOffset`.

4. **Mirror code-style của commit `baccd2c8`**: struct `RedisOffsetStore`, hàm `isStale`/`resolve*ID` trong processor, hàm `reclaim` trong consumer — giống hệt cách viết của commit gốc, chỉ khác key prefix và staleDuration.

## Risks / Trade-offs

- [Risk] Offset TTL quá ngắn (1m ontime) → sau khi offset hết hạn, message cũ được áp lại (safe vì idempotent handler, nhưng có thể ghi đè). → Mitigation: TTL đủ lớn hơn chu kỳ xử lý bình thường; handler vốn idempotent.
- [Risk] No-id message quay lại retry loop. → Mitigation: route sang DLQ permanent (deviation so với commit gốc, xem proposal).
- [Risk] `XAutoClaim` reclaim message đang được xử lý chậm (chưa tới idle) → không reclaim (chỉ > idle). An toàn.
- [Risk] Test reclaim phải chờ idle. → Mitigation: `reclaimIdleTime` là `var`, test set ~100ms.

## Migration Plan

- Chỉ thêm behaviour + key Redis mới; deploy không đổi schema. Rollback = revert commit (offset key sẽ dư thừa, tự hết TTL).
- Không cần migrate data.
