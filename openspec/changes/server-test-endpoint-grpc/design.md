## Context

`ping-service` là worker, chỉ chạy gRPC **client** (gọi `endpoint-service`), không có gRPC server. `PingWorker.Ping` trả `*Response{StatusCode, Body string}`, `ResponseChecker.CheckResponse` kết hợp status code + body expr (fail-safe). `server-service.TestEndpoint` (`service/endpoint.go:74`) hiện gọi `infrastructure.PingURL` — tự làm HTTP, chỉ check status code, bỏ qua body expr.

Mục tiêu: `server-service` test endpoint qua engine của `ping-service` để nhất quán. Vì 2 service là module Go riêng biệt, không thể import chung package → dùng gRPC.

## Goals / Non-Goals

**Goals:**
- `ping-service` expose gRPC `PingService.Ping`.
- `server-service.TestEndpoint` gọi qua gRPC, nhận `status_code` + `error`.
- Body check (`body_check_expr`) được áp dụng nhất quán.
- Error format phân biệt transport vs check failure.
- Xoá code ping trùng lặp (`pingclient.go`).

**Non-Goals:**
- Không thay đổi logic check của `ping-service` (chỉ expose thêm).
- Không thêm DB cho `ping-service` (build tạm `domain.Endpoint` từ request).
- Không đổi `ZSetWorkerRunner` poll flow.

## Decisions

1. **Proto mới `common/proto/ping/v1/ping_service.proto`**
   - `PingRequest{ method, url, timeout_ms, expected_code, body_check_expr }`
   - `PingResponse{ status_code int32, error string }` (error rỗng = ok)
   - `service PingService { rpc Ping(...) }`

2. **`PingServer` build tạm `domain.Endpoint`** từ request rồi gọi `PingWorker.Ping` + `ResponseChecker.CheckResponse`. ping-service không có DB → build tạm vô tư, reuse nguyên logic, không refactor `PingWorker`.

3. **Error format phân biệt 2 nguồn** (trong `PingServer`):
   - Ping/transport lỗi → `fmt.Sprintf("ping error: %s", err)` , `status_code = 0`.
   - Check thất bại (status/body) → `fmt.Sprintf("check failed: %s", err)`, giữ `status_code` thực tế.
   - Trả error qua field `Error` (không dùng `status.Error`) → client đọc trực tiếp business message.

4. **Port gRPC ping-service = 50053**, chỉ internal network (không traefik). `compose.yml` thêm port.

5. **`server-service` gRPC client** `PingClient` wrap `pingv1.NewPingServiceClient`, method `Ping(ctx, method, url, timeout, expectedCode, bodyExpr) (int, error)` map `PingResponse` → `(statusCode, err)`.

6. **Model + DTO + OpenAPI**: `domain.Endpoint` thêm `BodyCheckExpr *string`; `TestEndpointRequest`/`SetCheckMethodRequest` (qua `Endpoint` schema) thêm `body_check_expr`; regenerate ogen.

7. **Xoá `pingclient.go`** sau khi `TestEndpoint` chuyển sang gRPC.

## Risks / Trade-offs

- [Thêm gRPC server vào ping-service] → port + deployment surface. Mitigation: internal-only, không expose traefik.
- [Latency TestEndpoint +1 network hop] → acceptable (vốn sync user action).
- [ogen regen overwrite generated] → review diff, không mất custom.
- [PingWorker timeout] → lấy từ `timeout_ms` request qua tạm Endpoint, không đổi signature.

## Migration Plan

1. Thêm proto + regenerate.
2. ping-service: config + gRPC server + wiring + compose port.
3. server-service: config + client + model + DTO + OpenAPI regen + service + xoá pingclient.
4. DB: thêm cột `body_check_expr` nullable (endpoints table). Deploy order linh hoạt (nil = skip).
5. Rollback: revert server-service client → fallback không có (tạm thời mất test body) hoặc giữ `pingclient.go` provisional.

## Open Questions

<!-- none -->
