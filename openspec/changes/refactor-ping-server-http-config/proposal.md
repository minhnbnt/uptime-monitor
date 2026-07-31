## Why

Ping-service đang lưu cấu hình HTTP-DNS (port, endpoint_path, expected_code, body_check_expr, method) và `PingType` phẳng trên `domain.Server`, trong khi server-service (nguồn dữ liệu) đã tách chúng ra bảng/domain riêng `ServerHttpConfig` và suy ra ping mode từ việc record có tồn tại hay không. Bên cạnh đó, refactor `k8sclient` còn dang dở: `CheckPodStatus`/`PingCheck` cũ đã bị xóa nhưng gRPC server và ping loop vẫn tham chiếu API cũ nên ping-service không build được. Cần hoàn tất refactor cho khớp domain mới của server-service.

## What Changes

- **BREAKING (internal)**: `domain.Server` bỏ `PingType`, `Method`, `Port`, `EndpointPath`, `ExpectedCode`, `BodyCheckExpr`. Chỉ giữ k8s identity + `Interval`/`Timeout` + `HttpConfig *ServerHttpConfig` (nil = pod-status mode), theo đúng `server-service/internal/domain`.
- **Mới**: `domain.ServerHttpConfig` (Port, EndpointPath, ExpectedCode, BodyCheckExpr, Method) — chỉ các field ping-service thực sự dùng.
- **BREAKING (internal)**: Bỏ hẳn `PingType` ở domain/DTO; ping mode suy ra từ `HttpConfig != nil`.
- **Move**: gRPC server từ `internal/infrastructure/grpcserver/ping_server.go` → `internal/handler/ping_server.go` (theo pattern server-service), xóa package `grpcserver`.
- **Mới**: `service.UrlResolverService.ResolveURL` — thay thế `getURL` cũ đã xóa, ghép URL `http://<host>:<port>/<path>` từ `k8sclient.ResolveDomainName` + `dto.HttpCheckParams`.
- `service.ResponseChecker.CheckResponse` đổi signature nhận `dto.HttpCheckParams` thay vì `domain.Endpoint`.
- Cập nhật nguồn dữ liệu: `grpcclient/endpoint_client.go` map `http_dns_config` → `HttpConfig`; `redis/processor.go` cắt bỏ các field không còn tồn tại ở bảng `servers`; `scheduler/endpointcache.go` lưu/đọc http config (marker + prefixed fields).
- `service/pingloop.go` branch theo `sv.HttpConfig != nil`, dùng API mới `CheckObjectStatus` của `k8sclient.K8sClient`.

## Capabilities

### New Capabilities

None — pure internal refactor, không đổi hành vi gRPC/CDC/scheduler nhìn từ ngoài (`skip_specs: true`).

### Modified Capabilities

None.

## Impact

- `ping-service`: `internal/domain/endpoint.go`, `internal/domain/server_http_config.go` (mới), `internal/dto/pingParams.go`, `internal/service/pingloop.go`, `internal/service/url.go` (mới), `internal/service/responseChecker.go`, `internal/handler/ping_server.go` (mới), `internal/app/app.go`, `internal/app/grpc.go`, `internal/infrastructure/grpcclient/endpoint_client.go`, `internal/infrastructure/redis/processor.go`, `internal/infrastructure/scheduler/endpointcache.go`; xóa `internal/infrastructure/grpcserver/`.
- Test: `service/pingloop_test.go`, `service/responseChecker_test.go`; các test khác (processor, zsetloop, scheduler integration) không đổi.
- Không đổi proto, server-service, event-service.
