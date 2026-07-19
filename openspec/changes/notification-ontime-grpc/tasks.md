## 1. Proto & Codegen

- [ ] 1.1 Add `OntimeService` service + `GetServersOntime` RPC, and messages `GetServersOntimeRequest`, `GetServersOntimeResponse`, `ServerOntimeStat`, `OntimeDayStat` to `common/proto/event/v1/event_service.proto`
- [ ] 1.2 Run `buf generate` in `common/proto` to regenerate the `eventv1` package (client + server interfaces)

## 2. ontime-service gRPC Handler

- [ ] 2.1 Add public `GetServersOntime(ctx, userID uint) (map[uint][]dto.OntimeStats, error)` to `OntimeService` in `ontime-service/internal/service/ontime.go`, reusing the existing `getServersOntime` logic
- [ ] 2.2 Create `ontime-service/internal/handler/ontime_grpc.go` implementing `GetServersOntime` on a new `OntimeGRPCServer`, mapping results to `eventv1.ServerOntimeStat`/`OntimeDayStat`
- [ ] 2.3 Register `eventv1.RegisterOntimeServiceServer` on the existing `grpc.Server` in `ontime-service/internal/app/server.go`
- [ ] 2.4 Build `ontime-service` and confirm it compiles with the new RPC

## 3. notification-service gRPC Client Infra

- [ ] 3.1 Add `EventAddr` to `GRPCConfig` in `notification-service/internal/config/config.go` (plain `host:port`)
- [ ] 3.2 Add `GRPCOntimeClientWrapper` + `RegisterGRPCOntimeClient` in `notification-service/internal/config/grpc.go` (insecure, plain host:port), mirroring the existing server wrapper
- [ ] 3.3 Add `grpc.event_addr` default (`ontime-service:50052`) + env binding (`GRPC_EVENT_ADDR`) in `viper.go`
- [ ] 3.4 Add `grpc.event_addr: "ontime-service:50052"` to `notification-service/config.yaml` and `config/notification-service.yml`
- [ ] 3.5 Register `config.RegisterGRPCOntimeClient` in `notification-service/internal/app/injector.go`

## 4. Migrate Ontime Adapter to gRPC

- [ ] 4.1 Add `userID uint` parameter to `OntimeAdapter.GetServersOntimeForDates` in `notification-service/internal/service/digest.go` and update the call site (`SendReport`) to pass `userID`
- [ ] 4.2 Rewrite `notification-service/internal/infrastructure/ontimeclient/client.go` to hold `eventv1.OntimeServiceClient` and implement `GetServersOntimeForDates` via the `GetServersOntime` RPC (map response into `map[uint][]domain.OntimeStats`)
- [ ] 4.3 Keep debug/error logging on each gRPC call (request sent, failure with error and target)
- [ ] 4.4 Remove all HTTP/`net/http`/`encoding/json` code from the package
- [ ] 4.5 Build `notification-service` and confirm the `OntimeAdapter` interface is still satisfied

## 5. Verification

- [ ] 5.1 Run `go build ./...` for `ontime-service` and `notification-service`
- [ ] 5.2 Run `go test ./...` for both services
- [ ] 5.3 Trigger a digest and confirm `SendReport` reaches `buildReport` (no 404, no `Activity error`); revert temporary `log.level: debug` in `config/notification-service.yml` back to `info`
