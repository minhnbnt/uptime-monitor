# Loại bỏ `UpdateMonitorStatus` — Status & Plan

## Context

`monitor_status` trên `endpoints` table là dư thừa. Status hiện tại có thể suy ra từ event gần nhất trong `server_events` (do ontime-service quản lý). Ping-service không cần gửi gRPC `UpdateMonitorStatus` về server-service nữa.

---

## ✅ Part A — Đã hoàn thành (commit 79faad7)

### Ping-service (3 files)

| File | Thay đổi |
|---|---|
| `internal/infrastructure/grpcclient/endpoint_client.go` | Xóa method `UpdateMonitorStatus` (giữ `GetBatch`) |
| `internal/infrastructure/interfaces.go` | Xóa interface `EndpointStatusUpdater` |
| `internal/infrastructure/recordstatus.go` | Xóa field `endpointStatusUpdater`, import `grpcclient`, dòng gọi `UpdateMonitorStatus` |

### Proto (2 files)

| File | Thay đổi |
|---|---|
| `common/proto/endpoint/v1/endpoint_service.proto` | Xóa `UpdateMonitorStatusRequest`, `UpdateMonitorStatusResponse`, `rpc UpdateMonitorStatus` |
| `common/proto/generated/endpoint/v1/*.pb.go` | Regenerate với `buf generate` |

### Server-service gRPC handler (1 file)

| File | Thay đổi |
|---|---|
| `internal/grpc/endpoint_server.go` | Xóa handler `UpdateMonitorStatus`, xóa field `endpointRepo` (nil pointer), xóa import `domain` + `repository` không dùng |

### Server-service ping infrastructure (3 files)

| File | Thay đổi |
|---|---|
| `internal/features/ping/infrastructure/interfaces.go` | Xóa interface `EndpointStatusUpdater` |
| `internal/features/ping/infrastructure/recordstatus.go` | Xóa field `endpointStatusUpdater`, import `serverrepo`, dòng gọi `UpdateMonitorStatus` |
| `internal/features/ping/infrastructure/recordstatus_test.go` | Xóa `mockEndpointStatusUpdater` + 8 occurrences của field trong test cases |

### Server-service endpoint repository (1 file)

| File | Thay đổi |
|---|---|
| `internal/features/server/repository/endpoint.go` | Xóa method `UpdateMonitorStatus`, xóa import `apperrors` |

### Server-service interfaces & mocks (3 files)

| File | Thay đổi |
|---|---|
| `internal/features/server/service/interfaces.go` | Xóa `UpdateMonitorStatus` khỏi `EndpointRepository` interface |
| `internal/features/importer/service/mocks_test.go` | Xóa field + mock method `UpdateMonitorStatus` |
| `internal/features/server/service/mocks_test.go` | Xóa field + mock method `UpdateMonitorStatus` |

---

## ✅ Part B — Đã hoàn thành (session 2026-07-16)

### Proto — thêm RPC mới

| File | Thay đổi |
|---|---|
| `common/proto/event/v1/event_service.proto` | Thêm `GetCurrentStatuses` + `CountByStatus` RPC + messages |
| Regenerate với `buf generate` | ✅ |

### Ontime-service — implement handler

| File | Thay đổi |
|---|---|
| `internal/grpc/event_server.go` | Inject `*gorm.DB` (lấy từ `config.GORMWrapper`), thêm handler `GetCurrentStatuses` (DISTINCT ON) + `CountByStatus` (subquery count) |

### Server-service — thêm gRPC client + sửa repository

| File | Thay đổi |
|---|---|
| `internal/config/config.go` | Thêm `EventAddr string` vào `GRPCConfig` |
| `config.yml` | Thêm `grpc.event_addr: ontime:50052` |
| `internal/grpcclient/event_client.go` | **New** — gRPC client với `GetCurrentStatuses` (map[uint]Status) + `CountByStatus` (online/offline counts) |
| `internal/features/server/repository/server.go` | `CountByStatus`: query endpoint IDs từ DB → gọi gRPC `CountByStatus` |
| `internal/features/server/repository/server.go` | Xóa `enrichMonitorStatus` (dead code) |
| `internal/app/injector.go` | Thêm `grpcclient.RegisterEventClient` (trước `RegisterServerRepository`) |

### Backend — cleanup

| File | Thay đổi |
|---|---|
| `internal/config/gorm.go` | Xóa `&domain.ServerEvent{}` khỏi AutoMigrate |

### Ghi chú

- `domain.Endpoint.MonitorStatus` đã được tag `gorm:"-"` (server-service không đọc/ghi column này qua GORM nữa)
- `UpsertEndpoint` vẫn set `MonitorStatus: domain.StatusOff` nhưng GORM bỏ qua vì `gorm:"-"`
- Server-service query endpoint IDs từ DB local, chỉ gửi IDs qua gRPC — không transfer dữ liệu thừa
- CountByStatus query: `SELECT status, COUNT(*) FROM (DISTINCT ON ... server_events WHERE ...) GROUP BY status`
