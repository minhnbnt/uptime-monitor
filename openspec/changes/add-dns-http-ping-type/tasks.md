## 1. Proto changes

- [x] 1.1 Add `enum PingType { PING_TYPE_POD_STATUS = 0; PING_TYPE_HTTP_DNS = 1; }` to `endpoint_service.proto` and `ping_service.proto`
- [x] 1.2 Add `message HttpDnsConfig { int32 port = 1; string endpoint_path = 2; }` to both proto files
- [x] 1.3 Add `PingType ping_type = 9; optional HttpDnsConfig http_dns_config = 10;` to `EndpointData`
- [x] 1.4 Add `PingType ping_type = 6; optional HttpDnsConfig http_dns_config = 7;` to `PingRequest`
- [x] 1.5 Regenerate proto Go code: `cd common/proto && buf generate`

## 2. Server-service — domain + DB

- [x] 2.1 Add `Service` to the `kind` enum in `api/schemas/server.yaml` and `api/schemas/endpoint.yaml`
- [x] 2.2 Add optional `http_config` object (`port`, `endpoint_path`) to `ServerObject`, `CreateServerRequest`, `UpdateServerRequest` in `api/schemas/server.yaml`
- [x] 2.3 Add `http_config` to `TestEndpointRequest` in `api/schemas/endpoint.yaml`
- [x] 2.4 Regenerate OpenAPI code: `go generate ./...` or `go tool ogen` in server-service
- [x] 2.5 Create `domain.ServerHttpConfig` model (`internal/domain/server_http_config.go`) — `ServerID` (PK), `Port`, `EndpointPath`
- [x] 2.6 Create `dto.HttpConfig` DTO (`internal/dto/http_config.go`)
- [x] 2.7 Create `repository.ServerHttpConfigRepository` — CRUD for `server_http_configs` table (`internal/infrastructure/repository/http_config.go`)
- [x] 2.8 Add `HttpConfig` field to `dto.Server`, `dto.CreateServerRequest`, `dto.UpdateServerRequest`, `dto.TestEndpointRequest` in `internal/dto/server.go`

## 3. Server-service — service + handlers

- [x] 3.1 Update `ServerService.CreateServer` — if `http_config` is set, create `ServerHttpConfig` record
- [x] 3.2 Update `ServerService.UpdateServer` — upsert/delete `ServerHttpConfig`
- [x] 3.3 Update `ServerService.GetServer` / `ListServers` — load `HttpConfig` on read
- [x] 3.4 Map `http_config` in `CreateServer` HTTP handler (`internal/handler/server.go`)
- [x] 3.5 Map `http_config` in `UpdateServer` HTTP handler (`internal/handler/server.go`)
- [x] 3.6 Map `http_config` in `TestEndpoint` handler (`internal/handler/endpoint.go`)
- [x] 3.7 Map `PingType` + `HttpDnsConfig` to `EndpointData` in gRPC handler (`internal/handler/endpoint_server.go`)
- [x] 3.8 Map `http_config` in `ToAPIServer` mapping (`internal/handler/mapping.go`)
- [x] 3.9 Update `EndpointService.TestEndpoint` — pass `PingType` + `HttpDnsConfig` in `PingRequest`

## 4. Ping-service — data model + pipeline

- [x] 4.1 Add `PingType` (uint), `Port` (int), `EndpointPath` (string) fields to `domain.Server` (`internal/domain/endpoint.go`)
- [x] 4.2 Map from `EndpointData` in gRPC client (`internal/infrastructure/grpcclient/endpoint_client.go`)
- [x] 4.3 Cache new fields in `ServerMetaCache.SetMulti` and `mapToServer` (`internal/infrastructure/scheduler/endpointcache.go`)
- [x] 4.4 Add fields to `debeziumServerData` and `toDomain()` in CDC processor (`internal/infrastructure/redis/processor.go`)

## 5. Ping-service — k8s client HTTP-DNS logic

- [x] 5.1 Add `HttpDnsCheck` method to `K8sClient` interface (`internal/infrastructure/k8sclient/client.go`)
- [x] 5.2 Implement `checkService` — look up K8s Service, construct DNS `<name>.<namespace>.svc.cluster.local`, HTTP GET to `http://<dns>:<port>/<path>`
- [x] 5.3 Implement `checkPodHTTP` — get Pod IP from K8s API, HTTP GET to `http://<ip>:<port>/<path>`
- [x] 5.4 Implement `checkStatefulSetHTTP` — construct DNS `<name>-0.<name>.<namespace>.svc.cluster.local`, HTTP GET to `http://<dns>:<port>/<path>`
- [x] 5.5 Add `"Service"` case to the kind switch in `CheckPodStatus` — delegates to `checkService` when `PingType = HTTP_DNS`
- [x] 5.6 Update `CheckPodStatus` signature to accept `PingCheck` struct; dispatch to HTTP-DNS methods when `PingType != POD_STATUS`
- [x] 5.7 Integrate `BodyChecker` — validate response body expression when `body_check_expr` is set
- [x] 5.8 Integrate `ResponseChecker` — validate expected status code when `expected_code > 0`

## 6. Ping-service — dispatch

- [x] 6.1 Update `pingWorker` interface in `pingloop.go` to accept `PingType`, `HttpDnsConfig` (port, endpoint_path, kind for DNS resolution)
- [x] 6.2 Update `pingAndRecordServer` to pass new params to ping worker
- [x] 6.3 Update `PingServer.Ping` gRPC handler — read `PingType` + `HttpDnsConfig` from `PingRequest`, call k8s client accordingly

## 7. Build & verify

- [x] 7.1 Build common proto: `go build ./...` in `common/proto`
- [x] 7.2 Build server-service: `go build ./...` in `server-service`
- [x] 7.3 Build ping-service: `go build ./...` in `ping-service`
