## Tasks

### 1. Route soft-delete trong processor
- [x] Sửa `ontime-service/internal/infrastructure/consumer/processor.go`: trong `onUpdate`, nếu `event.After.DeletedAt != nil` thì gọi `handler.OnDelete`, ngược lại `handler.OnUpdate` (không truyền `deletedAt`).

### 2. Gỡ `deletedAt` khỏi interface và service
- [x] `ontime-service/internal/infrastructure/consumer/consumer.go`: `ServerOwnerHandler.OnUpdate` bỏ param `deletedAt *time.Time`.
- [x] `ontime-service/internal/service/ownership_service.go`: `OnUpdate` bỏ param, gọi `s.repo.Upsert(ctx, serverID, userID)`; `OnCreate` gọi `s.repo.Upsert(ctx, serverID, userID)`; xoá import `time`.

### 3. Đơn giản hoá Upsert và làm Delete idempotent
- [x] `ontime-service/internal/infrastructure/repository/server_owner.go`: `Upsert` bỏ param `deletedAt` và khối set `DeletedAt`.
- [x] `server_owner.go`: `Delete` trả `nil` khi `rowAffected == 0` (chỉ báo lỗi khi `err != nil`).

### 4. Dọn dẹp lo.MapToSlice
- [x] `ontime-service/internal/handler/ontime_grpc.go`: `GetServersOntime` thay `lo.Map(lo.Keys(ontimeMap), ...)` bằng `lo.MapToSlice(ontimeMap, ...)`.

### 5. Verify
- [x] `cd ontime-service && go build ./...` thành công.
- [x] `golangci-lint run` trên các package đổi (consumer, service, repository, handler) = 0 issue.
- [x] Chạy test ontime-service (nếu có) xanh.
- [x] Commit theo quy ước repo.
