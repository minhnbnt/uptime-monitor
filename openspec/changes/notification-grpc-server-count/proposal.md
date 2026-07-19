## Why

Notification-service is the only internal service that talks to server-service over HTTP, while all other internal services (importer, ontime) already use gRPC (`serverv1.ServerServiceClient`). This inconsistency adds an unnecessary HTTP client, breaks the service-to-service convention, and contributes to the digest failures we saw (the missing `/api/v1/servers/count` HTTP route caused 404s and infinite Temporal retries). Moving notification-service onto gRPC aligns it with the rest of the system and removes the dead HTTP dependency.

The HTTP endpoint `/api/v1/servers/count` on server-service is kept intact, because it is the user-facing contract served through the API gateway (external clients rely on it). Only notification-service's internal call path changes.

## What Changes

- Add a new gRPC RPC `CountServersByStatus` to `server.v1.ServerService` (proto + generated code + server-side handler).
- Add a gRPC client to notification-service that dials server-service over gRPC.
- Convert notification-service's `ServerAdapter` (`List` and `CountByStatus`) from HTTP calls to gRPC calls.
- Remove the now-unused HTTP `serverclient` (base URL, `net/http`, JSON decode).
- Keep server-service's HTTP `GET /api/v1/servers/count` endpoint and its ogen-generated code unchanged.

## Capabilities

### New Capabilities
- `server-grpc-count`: Exposes a gRPC `CountServersByStatus(user_id)` RPC on server-service returning `{total, online, offline}`, mirroring the existing HTTP `/api/v1/servers/count` behavior.
- `notification-server-grpc`: notification-service consumes server-service exclusively over gRPC (ListServers + CountServersByStatus), replacing its HTTP server client.

### Modified Capabilities
<!-- No existing openspec/specs capabilities are changing at the requirement level; this is a transport switch for an internal client. -->

## Impact

- `common/proto/server/v1/server_service.proto` and generated `serverv1` package (regenerated via `buf generate`).
- `server-service/internal/handler/server_server.go`: new `CountServersByStatus` gRPC handler (reuses existing `ServerService.CountByStatus`).
- `notification-service`: new `config/grpc.go` + gRPC client wiring; rewritten `internal/infrastructure/serverclient/client.go`; updated `config.go` (add `grpc.server_addr`), `config.yaml`, `viper.go` defaults; `app/injector.go` registration.
- No public API changes; HTTP `/api/v1/servers/count` stays for the gateway.
- Backward compatible: adding an RPC is non-breaking for existing gRPC callers (importer, ontime).
