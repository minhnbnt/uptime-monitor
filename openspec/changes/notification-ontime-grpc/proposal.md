## Why

The digest workflow in notification-service fails because its internal ontime client calls `POST /api/v1/servers/ontime/batch`, an endpoint that does not exist in ontime-service (404). The real ontime HTTP endpoint `GET /api/v1/servers/ontime` requires a JWT (enforced by the API gateway's forward-auth), so it cannot be used for service-to-service calls. This 404 collapses into `ErrInternal` and triggers infinite Temporal retries, breaking digest report generation.

ontime-service already exposes gRPC on `:50052`, but only `GetCurrentStatuses` and `CountByStatus` (current status, no daily uptime). The per-day uptime statistics the digest needs are computed internally (`OntimeService.getServersOntime`) but never exposed over gRPC. We must expose them via a new gRPC RPC so internal callers can fetch daily ontime without HTTP/JWT.

## What Changes

- Add a new gRPC service `OntimeService` (in `event.v1`) to ontime-service exposing `GetServersOntime(user_id)` returning per-server daily `ontime_stats` for the last 30 days.
- Implement the handler on ontime-service, reusing the existing `OntimeService.getServersOntime` computation (no new business logic).
- Add a gRPC client to notification-service that dials ontime-service over gRPC (`ontime-service:50052`).
- Convert notification-service's `OntimeAdapter` (`GetServersOntimeForDates`) from HTTP to gRPC.
- Remove the dead HTTP ontime client (`net/http`, JSON encode/decode, `/batch` URL).
- The HTTP `GET /api/v1/servers/ontime` endpoint on ontime-service remains unchanged for external/gateway clients.

## Capabilities

### New Capabilities
- `notification-ontime-grpc`: notification-service consumes ontime statistics from ontime-service exclusively over gRPC (no HTTP/JWT), replacing its broken HTTP ontime client.
- `ontime-grpc-stats`: ontime-service exposes a gRPC `GetServersOntime` RPC returning per-server daily uptime statistics for a user.

### Modified Capabilities
<!-- No existing openspec/specs capabilities are changing at the requirement level; this is a new transport + new RPC for an internal client. -->

## Impact

- `common/proto/event/v1/event_service.proto` and generated `eventv1` package (regenerated via `buf generate`).
- `ontime-service/internal/handler/`: new gRPC handler; `app/server.go` registers the new service on the existing `:50052` gRPC server.
- `ontime-service/internal/service/ontime.go`: new public `GetServersOntime` method wrapping the existing private `getServersOntime`.
- `notification-service`: new `config` gRPC wiring for ontime (`EventAddr`/`ontime-service:50052`); rewritten `internal/infrastructure/ontimeclient/client.go`; updated `config.go`, `viper.go`, `config.yml`, `app/injector.go`; `OntimeAdapter` interface gets a `userID` parameter.
- No public API changes; HTTP `GET /api/v1/servers/ontime` stays for the gateway.
- Backward compatible: adding an RPC is non-breaking for existing gRPC callers (ping-service uses `EventRecorderService`/`StatusService`).
