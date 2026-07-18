## 1. API Schema & Codegen

- [x] 1.1 Add `monitor_status` (nullable string enum `ON`/`OFF`) to `ServerObject` in `server-service/api/schemas/server.yaml`
- [x] 1.2 Regenerate ogen types in `server-service` (`go generate ./...`) and verify `generated/api` includes `MonitorStatus` (`OptString`)

## 2. DTO

- [x] 2.1 Add `MonitorStatus domain.ServerStatus` field to `dto.Server` in `server-service/internal/dto/server.go` (leave zero-value in `ServerFromDomain`)

## 3. Service Layer Enrichment

- [x] 3.1 Define/confirm a `StatusClient` interface (reuse `grpcclient.StatusClient`) and inject it into `ServerService` (`RegisterServerService` in `server-service/internal/app/injector.go`)
- [x] 3.2 Add a private helper `applyStatuses(ctx, servers []dto.Server)` that collects endpoint IDs, calls `GetCurrentStatuses` once, and maps results back by `server.Endpoint.ID` (guard nil endpoint)
- [x] 3.3 On `GetCurrentStatuses` error: log warning and return without status (best-effort)
- [x] 3.4 Call `applyStatuses` in `ListServers`, `GetServer`, and `SearchServers` after the server list is resolved

## 4. Handler Mapping

- [x] 4.1 In `server-service/internal/handler/mapping.go` `ToAPIServer`, set `MonitorStatus: api.NewOptString(string(s.MonitorStatus))`

## 5. Verification

- [x] 5.1 Build `server-service` (`go build ./...`) and `go vet ./...`
- [x] 5.2 Confirm `monitor_status` appears in List/Get/Search responses and is `null` for servers with no endpoint/events
