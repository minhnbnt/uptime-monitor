## 1. Proto

- [x] 1.1 Tạo `common/proto/ping/v1/ping_service.proto` với `PingRequest`, `PingResponse`, `service PingService`
- [x] 1.2 Regenerate proto (`buf generate`) → `common/proto/generated/ping/v1`

## 2. ping-service gRPC server

- [x] 2.1 `config.go`: `GRPCConfig` thêm `ServerPort` (`server_port`); `config/ping-service.yml` thêm `grpc.server_port: "50053"`
- [x] 2.2 Tạo `internal/infrastructure/grpcserver/ping_server.go`: implement `PingServiceServer`, build tạm `domain.Endpoint`, gọi `PingWorker.Ping` + `ResponseChecker.CheckResponse`, wrap error (`ping error:` / `check failed:`)
- [x] 2.3 `internal/app/server.go`: thêm `RunPingGRPCServer`; `cmd/main.go`: gọi trong waitgroup
- [x] 2.4 `compose.yml`: ping-service thêm port `50053`

## 3. server-service gRPC client

- [x] 3.1 `config.go`: `GRPCConfig` thêm `PingAddr` (`ping_addr`); `config/server-service.yml` thêm `grpc.ping_addr: "ping-service:50053"`
- [x] 3.2 Tạo `internal/infrastructure/grpcclient/ping_client.go`: `PingClient` wrap `pingv1.NewPingServiceClient`, method `Ping(...) (int, error)`

## 4. server-service model + DTO + OpenAPI

- [x] 4.1 `domain/server.go`: `Endpoint` thêm `BodyCheckExpr *string`
- [x] 4.2 `api/schemas/endpoint.yaml`: thêm `body_check_expr` vào `Endpoint` schema và `TestEndpointRequest`
- [x] 4.3 Regenerate ogen (`go generate ./...` trong server-service), review diff `generated/api`
- [x] 4.4 `dto/server.go`: `SetCheckMethodRequest` + `TestEndpointRequest` thêm `BodyCheckExpr string`
- [x] 4.5 `service/endpoint.go`: `toDomainEndpoint` map field; `TestEndpoint` gọi `pingClient.Ping` thay `PingURL`

## 5. Cleanup & verify

- [x] 5.1 Xoá `server-service/internal/infrastructure/pingclient.go`
- [x] 5.2 Build + test cả ping-service và server-service (`go build ./...`, `go test ./...`, `go vet ./...`)
