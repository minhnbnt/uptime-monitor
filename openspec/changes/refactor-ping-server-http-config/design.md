## Context

Refactor này nằm gọn trong `ping-service`. Trước khi làm việc này, `k8sclient` đã được tách (xem diff working tree) thành `K8sClient` với `CheckObjectStatus(ctx, *dto.K8sObjectCheckParams)` và `ResolveDomainName(ctx, *dto.K8sObjectCheckParams)`, xóa `PingCheck`/`CheckPodStatus` cũ — nhưng `pingloop.go` và gRPC server vẫn gọi API cũ nên `go build ./...` đang fail. Đồng thời `domain.Server` của ping-service đang mang cả cấu hình HTTP phẳng, lệch với `server-service/internal/domain` (Server = k8s identity; ServerHttpConfig riêng; ping mode suy từ sự tồn tại của record config).

Nguồn dữ liệu của ping-service:
- **gRPC `GetEndpoints`** (server-service → `grpcclient/endpoint_client.go`): `EndpointData` có `ping_type` + `http_dns_config`; đây là nguồn đầy đủ duy nhất — `ServerProvider.GetBatch` fetch gRPC khi cache miss và `SetMulti` vào cache.
- **CDC Debezium** (`redis/processor.go`, stream `uptime.public.servers`): chỉ chứa row bảng `servers` (identity + interval/timeout), KHÔNG có http config hay ping_type. CDC chỉ dùng để `Register`/`Unregister` vào zset và `Delete` cache (server-service đã xóa cache khi update → lần sau fetch gRPC lại).

## Goals / Non-Goals

**Goals:**
- `domain.Server` của ping-service khớp server-service: chỉ k8s identity + Interval/Timeout + `HttpConfig *ServerHttpConfig`; bỏ `PingType` và các field http phẳng.
- Hoàn tất refactor k8sclient: ping loop + gRPC handler dùng API mới; build/vet/unit test xanh.
- Move gRPC server vào `internal/handler`, xóa package `grpcserver`.

**Non-Goals:**
- Đổi proto, server-service, event-service, ontime-service.
- Không đổi hành vi bên ngoài (gRPC Ping, CDC, scheduler) — `skip_specs: true`.
- Không thêm interface mới cho `K8sClient` (giữ concrete struct như refactor đã định); chỉ thêm interface cục bộ `urlResolver` trong `pingloop.go` để test được nhánh HTTP.

## Decisions

### D1: ServerHttpConfig — chỉ field cần thiết
- **Decision**: `domain.ServerHttpConfig{Port, EndpointPath, ExpectedCode, BodyCheckExpr, Method}` — 5 field ping-service dùng trong HTTP-DNS check. Bỏ `ServerID` (đã có trong `Server`, quan hệ 1-1), bỏ timestamps, bỏ gorm tags.
- **Rationale**: ping-service không persist domain; giữ tối thiểu field đúng yêu cầu "chỉ dùng field cần thiết".
- **Alternative**: mirror y hệt server-service (thêm ServerID + CreatedAt/UpdatedAt) — rejected: field chết, vô dụng.

### D2: Bỏ PingType, suy từ HttpConfig
- **Decision**: `domain.Server` và `dto.CheckParams` không còn `PingType`. Ping loop + handler quyết định mode từ `sv.HttpConfig != nil`.
- **Rationale**: khớp server-service (không có cột ping_type, `EndpointData.PingType` do server-service tính từ config). Loại bỏ nguồn lệch dữ liệu (PingType cũ lưu trong CDC cache sẽ không còn nguồn).
- **Alternative**: giữ PingType trên Server — rejected vì phải đoán/bảo trì từ dữ liệu không có nguồn đáng tin (CDC không gửi).

### D3: Cache lưu http config qua marker
- **Decision**: `ServerMetaCache.SetMulti` lưu field base (namespace, kind, object_id, container_name, interval_ns, timeout_ns). Khi `HttpConfig != nil` thêm marker `http_config="1"` + `http_port`, `http_endpoint_path`, `http_expected_code`, `http_body_check_expr`, `http_method`. `mapToServer` dựng `HttpConfig` khi marker có mặt.
- **Rationale**: cache luôn được fill từ gRPC (đầy đủ config). Nếu không lưu config, một server HTTP_DNS trúng cache sẽ bị ping nhầm POD_STATUS. OnUpdate xóa cache → refetch gRPC nên không lo stale khi config đổi/bị xóa.
- **Alternative**: không lưu http config trong cache — rejected vì phá cache-hit của server HTTP_DNS.

### D4: UrlResolverService ở tầng service
- **Decision**: `service.UrlResolverService.ResolveURL(ctx, *dto.K8sObjectCheckParams, *dto.HttpCheckParams) (string, error)` — gọi `K8sClient.ResolveDomainName` rồi ghép `http://<host>:<port>/<path>`. Là thay thế trực tiếp cho `getURL` cũ đã xóa khỏi `http_dns.go`.
- **Rationale**: `k8sclient` chỉ lo k8s; việc ghép URL + HTTP ping nằm ở tầng service. Gộp port/path vào một chỗ thay vì mỗi caller tự format.
- **Alternative**: để logic URL trong handler + ping loop mỗi nơi — rejected: trùng lặp.

### D5: ResponseChecker nhận dto.HttpCheckParams
- **Decision**: `CheckResponse(http *dto.HttpCheckParams, resp infra.Response) error`. Giữ nguyên semantics hiện có: `ExpectedCode <= 0` → không check status; `BodyCheckExpr == ""` → không check body (test hiện có assert cả 2).
- **Rationale**: `dto.HttpCheckParams` đã là shape chung (UrlResolverService cùng dùng); tránh phụ thuộc domain. Signature cũ nhận `domain.Endpoint` vỡ vì Server không còn field http.
- **Lưu ý hành vi**: khác `checkOK` cũ (cũ yêu cầu 2xx khi expected_code=0); đây là hành vi có chủ đích của refactor trước (test "default no check" assert 500 → no error), giữ nguyên.

### D6: gRPC handler move + DI
- **Decision**: `internal/handler/ping_server.go` — struct `PingServer` (embed `pingv1.UnimplementedPingServiceServer`), `RegisterPingServer(i do.Injector)`, inject `*k8sclient.K8sClient`, `*service.UrlResolverService`, `*infrastructure.PingClient`, `*service.ResponseChecker`. `app/grpc.go` đổi sang `*handler.PingServer`. Xóa `internal/infrastructure/grpcserver/`.
- **Rationale**: theo pattern `handler/*_server.go` của server-service (server_server.go, endpoint_server.go). Ping handler giữ nguyên hành vi cũ: trả `Running:false, Error:"check error: ..."` thay vì gRPC error; `HttpDnsConfig == nil` → error rõ ràng.

### D7: CDC processor cắt field không còn tồn tại
- **Decision**: `debeziumServerData` chỉ giữ ID, Namespace, Kind, ObjectID, ContainerName, Interval, Timeout; `toDomain()` map tương ứng. Bỏ `PingType`, `Port`, `EndpointPath`, `ExpectedCode`, `BodyCheckExpr`, `Method`.
- **Rationale**: bảng `servers` không còn các cột đó (đã sang `server_http_configs`); CDC chỉ cần ID + Interval cho `Register` (zset).

## Risks / Trade-offs

- **[Cache chứa config cũ]** Nếu config đổi/xóa mà CDC `servers` không phát event (config nằm bảng khác) → cache có thể serve config cũ → Mitigation: server-service xóa cache (`OnUpdate`) và ping-service luôn fetch gRPC khi cache miss; cache là best-effort, sai lệch tự hết sau TTL 1h.
- **[Test phủ nhánh HTTP_DNS]** Nhánh HTTP trong ping loop khó test vì `UrlResolverService`/`K8sClient` là concrete → Mitigation: thêm interface cục bộ `urlResolver` trong `pingloop.go` để mock; `PingClient` test bằng `httptest.Server`.
- **[Hành vi expected_code=0]** ResponseChecker chấp nhận mọi status khi `ExpectedCode=0` (khác `checkOK` cũ yêu cầu 2xx) → Mitigation: đây là hành vi có chủ đích, có test assert sẵn; ghi chú lại để không ai "fix" ngược.

## Migration Plan

- Internal refactor, không cần migration dữ liệu / deploy phối hợp: cache redis đã có TTL, tự refresh từ gRPC.
- Rollback: revert working tree; không có thay đổi proto hay DB.

## Open Questions

None.
