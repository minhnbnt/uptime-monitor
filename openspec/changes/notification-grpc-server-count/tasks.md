## 1. Proto & Codegen

- [x] 1.1 Add `CountServersByStatusRequest`/`CountServersByStatusResponse` messages and the `CountServersByStatus` RPC to `common/proto/server/v1/server_service.proto`
- [x] 1.2 Run `buf generate` in `common/proto` to regenerate the `serverv1` package (client + server interfaces)

## 2. Server-side gRPC Handler

- [x] 2.1 Implement `CountServersByStatus` on `*ServerServer` in `server-service/internal/handler/server_server.go`, mapping to `serverService.CountByStatus(ctx, uint(req.UserId))`
- [x] 2.2 Build `server-service` and confirm it compiles with the new RPC

## 3. Notification-service gRPC Client Infra

- [x] 3.1 Add `GRPC GRPCConfig` (with `ServerAddr`) to `notification-service/internal/config/config.go`
- [x] 3.2 Create `notification-service/internal/config/grpc.go` with `GRPCClientWrapper` + `RegisterGRPCClient` (plain `host:port`, insecure), mirroring ontime-service
- [x] 3.3 Add `grpc.server_addr` default + env binding in `viper.go` (default `localhost:50051`)
- [x] 3.4 Add `grpc.server_addr: "localhost:50051"` to `notification-service/config.yaml`
- [x] 3.5 Register `config.RegisterGRPCClient` in `notification-service/internal/app/injector.go`

## 4. Migrate Server Adapter to gRPC

- [x] 4.1 Rewrite `notification-service/internal/infrastructure/serverclient/client.go` to hold `serverv1.ServerServiceClient` and implement `List` (via `ListServers` RPC) and `CountByStatus` (via `CountServersByStatus` RPC)
- [x] 4.2 Keep debug/error logging on each gRPC call (request sent, failure with error)
- [x] 4.3 Remove all HTTP/`net/http`/`encoding/json` code from the package
- [x] 4.4 Update the client constructor/registration to use `config.GRPCClientWrapper` instead of HTTP base URL
- [x] 4.5 Build `notification-service` and confirm `ServerAdapter` interface is still satisfied

## 5. Verification

- [x] 5.1 Run `go build ./...` for `server-service` and `notification-service`
- [x] 5.2 Run `go test ./...` for both services
- [x] 5.3 Confirm HTTP `GET /api/v1/servers/count` still exists in server-service generated/ogen code (unchanged)
