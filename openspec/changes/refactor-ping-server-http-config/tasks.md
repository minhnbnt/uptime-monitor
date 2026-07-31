## 1. Domain & DTO

- [x] 1.1 `internal/domain/endpoint.go` — `Server` chỉ giữ ID, Namespace, Kind, ObjectID, ContainerName, Interval, Timeout, `HttpConfig *ServerHttpConfig`; xóa PingType/Method/Port/EndpointPath/ExpectedCode/BodyCheckExpr; xóa alias `Endpoint`
- [x] 1.2 `internal/domain/server_http_config.go` — MỚI: `ServerHttpConfig{Port int; EndpointPath string; ExpectedCode int; BodyCheckExpr string; Method string}`
- [x] 1.3 `internal/dto/pingParams.go` — bỏ `PingType` khỏi `CheckParams`

## 2. Nguồn dữ liệu

- [x] 2.1 `internal/infrastructure/grpcclient/endpoint_client.go` — map `http_dns_config` → `sv.HttpConfig` (bỏ `PingType`)
- [x] 2.2 `internal/infrastructure/redis/processor.go` — `debeziumServerData` + `toDomain()` chỉ còn identity + interval/timeout
- [x] 2.3 `internal/infrastructure/scheduler/endpointcache.go` — `SetMulti` lưu marker `http_config=1` + `http_port/http_endpoint_path/http_expected_code/http_body_check_expr/http_method` khi `HttpConfig != nil`; `mapToServer` dựng `HttpConfig` từ marker

## 3. Service layer

- [x] 3.1 `internal/service/url.go` — MỚI: `UrlResolverService` (inject `*k8sclient.K8sClient`), `ResolveURL(ctx, *dto.K8sObjectCheckParams, *dto.HttpCheckParams) (string, error)`
- [x] 3.2 `internal/service/responseChecker.go` — `CheckResponse(http *dto.HttpCheckParams, resp infra.Response) error`; giữ semantics (ExpectedCode<=0 bỏ qua status, BodyCheckExpr=="" bỏ qua body)
- [x] 3.3 `internal/service/pingloop.go` — `pingWorker` interface = `CheckObjectStatus(ctx, *dto.K8sObjectCheckParams)`; thêm dep urlResolver (interface cục bộ), `*PingClient`, `*ResponseChecker`; `checkServer` branch `sv.HttpConfig != nil` → `checkHTTPDNS` (build `dto.HttpCheckParams` từ HttpConfig) else `CheckObjectStatus`; sửa `do.MustInvoke[*k8sclient.K8sClient]`

## 4. gRPC handler (move)

- [x] 4.1 `internal/handler/ping_server.go` — MỚI: `PingServer` (embed `pingv1.UnimplementedPingServiceServer`), inject `*k8sclient.K8sClient`, `*service.UrlResolverService`, `*infrastructure.PingClient`, `*service.ResponseChecker`; `Ping` branch theo `req.PingType` (POD_STATUS → `CheckObjectStatus`, HTTP_DNS → resolve URL + ping + `CheckResponse`); giữ hành vi trả `Running:false, Error:"check error: ..."`; `HttpDnsConfig==nil` → error
- [x] 4.2 Xóa `internal/infrastructure/grpcserver/ping_server.go` + package `grpcserver`
- [x] 4.3 `internal/app/app.go` — bỏ `pinggrpcserver.RegisterPingServer`, thêm `pinghandler.RegisterPingServer` + `pingservice.RegisterUrlResolverService`
- [x] 4.4 `internal/app/grpc.go` — invoke `*handler.PingServer`

## 5. Tests & verify

- [x] 5.1 `internal/service/responseChecker_test.go` — chuyển sang `dto.HttpCheckParams` (bỏ `strptr`/`domain.Endpoint`)
- [x] 5.2 `internal/service/pingloop_test.go` — mock `CheckObjectStatus` + deps mới; thêm 1 test nhánh HTTP_DNS (mock `urlResolver` + `httptest.Server`)
- [x] 5.3 `go build ./...` và `go vet ./...` trong `ping-service` không lỗi
- [x] 5.4 `go test ./internal/service/ ./internal/handler/ ./internal/dto/ ./internal/infrastructure/redis/` xanh (bỏ integration cần docker)
