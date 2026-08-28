## Tổng quan

Áp dụng ý tưởng từ commit `fd8d83b` (nhánh k8s) vào `master`, chỉ lấy phần hành vi (behavioral) khớp với code hiện tại. Nhánh k8s có thêm khung `RedisOffsetStore`/idempotency — phần đó **không có ở master** nên nằm ngoài scope.

## Quyết định thiết kế

1. **Route soft-delete trong `processor.go`**
   - `onUpdate`: nếu `event.After != nil` và `event.After.DeletedAt != nil` → `handler.OnDelete(ctx, event.After.ID)`; ngược lại → `handler.OnUpdate(ctx, event.After.ID, event.After.CreatedByID)`.
   - Giữ nguyên `op` matching `c/r/u/d` (đã có sẵn ở `processor.go`).

2. **Gỡ `deletedAt` khỏi `OnUpdate`**
   - Interface `ServerOwnerHandler.OnUpdate` (consumer.go) và implement `OwnershipService.OnUpdate` (ownership_service.go) bỏ param `deletedAt *time.Time`.
   - `OwnershipService.OnUpdate` gọi `repo.Upsert(ctx, serverID, userID)`.

3. **Đơn giản hoá `Upsert`**
   - `ServerOwnerRepository.Upsert` bỏ param `deletedAt *time.Time` và khối set `DeletedAt`. Chỉ còn `Save(&owner)`.
   - Cập nhật 2 caller (`OnCreate`, `OnUpdate`) trong `ownership_service.go`. Bỏ import `time` ở file đó.

4. **`Delete` idempotent**
   - `ServerOwnerRepository.Delete`: sau `Delete(ctx)`, nếu `err != nil` mới báo lỗi; ngược lại luôn `return nil` (kể cả `rowAffected == 0`).
   - `ServerOwner` có `gorm.DeletedAt` nên `Delete` của gorm là soft-delete (set `deleted_at`), nhất quán với语义 hiện tại.

5. **`lo.MapToSlice` trong `GetServersOntime`**
   - `ontime_grpc.go`: thay `lo.Map(lo.Keys(ontimeMap), func(id, _) ...)` bằng `lo.MapToSlice(ontimeMap, func(id uint, stats []dto.OntimeStats) ...)`. Giữ nguyên truy cập `stat.Date`/`stat.Stats` (dto hiện tại khác với commit gốc).

## Rủi ro / lưu ý

- Việc route soft-delete vào `Delete` (thay vì `Upsert` set `deleted_at`) về kết quả đều gắn `deleted_at` vào `server_owners`, nên `CountOwnedServers`/`GetOwnedServers` (lọc `deleted_at IS NULL`) vẫn loại bỏ đúng.
- `Delete` idempotent bằng soft-delete: event xoá trùng sẽ gặp dòng đã bị soft-delete (gorm lọc `deleted_at IS NULL`) → `rowAffected == 0` → no-op. Đúng ý định.
- Không đổi API gRPC/public, không đổi schema DB → không breaking.

## Ngoài scope

- Tách `debeziumMessage.go` (file split) — để sau nếu cần.
- Khung offset/idempotency (`RedisOffsetStore`) — không có ở master.
- Wire `domain.ToServerStatus` — hàm đã tồn tại, chưa dùng đến ở luồng này.
