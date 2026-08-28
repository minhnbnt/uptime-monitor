## Why

Consumer Debezium của ontime-service xử lý CDC từ bảng `servers` để duy trì bảng `server_owners` (áp dụng quyền sở hữu server). Hiện tại:

- Một sự kiện UPDATE có `deleted_at` được set (soft-delete ở nguồn) vẫn đi qua `OnUpdate` và ghi `deleted_at` vào `server_owners` qua `Upsert` — đường xử lý không nhất quán với sự kiện DELETE thật.
- `ServerOwnerRepository.Delete` báo lỗi khi không có dòng nào bị xoá (`rowAffected == 0`), nên một sự kiện xoá được gửi lại (duplicate/redelivery của Redis Stream) sẽ làm consumer fail và không ack → lặp lại vô tận.

Ý tưởng được lấy từ commit `fd8d83b` (nhánh k8s): route soft-delete vào `OnDelete` và làm `Delete` thành idempotent. Mục tiêu là consumer xử lý đúng và khoẻ hơn với event trùng lặp, mà không cherry-pick toàn bộ commit (nhánh k8s có thêm khung offset/idempotency không có ở master).

## What Changes

- **Route soft-delete**: trong `processor.go`, sự kiện `u` có `after.deleted_at != nil` sẽ gọi `OnDelete` thay vì `OnUpdate`.
- **Bỏ tham số `deletedAt`** khỏi `OnUpdate` (interface ở `consumer.go` và implement ở `ownership_service.go`) vì soft-delete giờ chuyển sang `Delete`.
- **`Upsert` đơn giản hoá**: bỏ tham số `deletedAt` (luôn là `nil` sau thay đổi trên), chỉ còn upsert `server_id`/`user_id`.
- **`Delete` idempotent**: trả `nil` khi `rowAffected == 0` (event xoá trùng/lặp là no-op, không báo lỗi).
- **Dọn dẹp `lo.MapToSlice`**: trong `ontime_grpc.go` `GetServersOntime`, thay `lo.Map(lo.Keys(ontimeMap), ...)` bằng `lo.MapToSlice(ontimeMap, ...)`.

## Capabilities

### New Capabilities
- `server-ownership`: hành vi của ownership consumer — ánh xạ CDC event (create/snapshot/update/delete) từ bảng `servers` sang bảng `server_owners`, bao gồm route soft-delete vào xoá và tính idempotent của thao tác xoá.

### Modified Capabilities
- (không có)

## Impact

- `ontime-service/internal/infrastructure/consumer/processor.go` — `onUpdate` route soft-delete.
- `ontime-service/internal/infrastructure/consumer/consumer.go` — interface `OnUpdate` bỏ `deletedAt`.
- `ontime-service/internal/service/ownership_service.go` — `OnUpdate`/`OnCreate` gọi `Upsert` không có `deletedAt`; bỏ import `time`.
- `ontime-service/internal/infrastructure/repository/server_owner.go` — `Upsert` bỏ `deletedAt`, `Delete` idempotent.
- `ontime-service/internal/handler/ontime_grpc.go` — `GetServersOntime` dùng `lo.MapToSlice`.
- Không đổi API gRPC/public, không đổi schema DB.
