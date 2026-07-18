## Why

`server-service.TestEndpoint` hiện tự thực hiện HTTP ping qua `infrastructure.PingURL`, chỉ kiểm tra status code. Trong khi đó `ping-service` đã có engine check đầy đủ (`PingWorker` + `ResponseChecker`) bao gồm cả body check bằng expr-lang. Khi user cấu hình `body_check_expr` trên endpoint, `TestEndpoint` sẽ không validate body → behaviour lệch với ping thực tế. Cần cho `server-service` dùng chung engine ping của `ping-service` để nhất quán.

## What Changes

- Thêm gRPC service mới `PingService` trong `ping-service` (RPC `Ping`), expose port riêng (50053).
- `server-service` thêm gRPC client gọi `ping-service.Ping` thay vì tự `PingURL`.
- `PingRequest`/`TestEndpointRequest`/`SetCheckMethodRequest`/`Endpoint` model thêm trường `body_check_expr`.
- `PingServer` trả `PingResponse{ status_code, error }` (error rỗng = ok), phân biệt `ping error:` (transport) và `check failed:` (status/body không khớp).
- Xoá `server-service/internal/infrastructure/pingclient.go` (không còn dùng).

## Capabilities

### New Capabilities
- `server-ping-grpc`: `server-service` gọi `ping-service` qua gRPC `PingService.Ping` để test endpoint, dùng chung engine check (status code + body expr), bao gồm error format chuẩn.

### Modified Capabilities
<!-- none -->

## Impact

- `common/proto/ping/v1/ping_service.proto`: service + message mới (regenerate).
- `ping-service`: config (grpc.server_port), gRPC server impl, app wiring, compose port.
- `server-service`: config (grpc.ping_addr), gRPC client, domain model, DTO, OpenAPI schema + regenerate (ogen), service `TestEndpoint`, xoá `pingclient.go`.
- Không phải breaking change (proto/service mới, cột DB nullable, DTO field optional).
