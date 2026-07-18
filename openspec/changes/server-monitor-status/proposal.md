## Why

The `server-service` API no longer exposes the live monitoring status of a server, even though the uptime data already lives in `ontime-service` (served via the newly split `StatusService.GetCurrentStatuses` gRPC method). Clients must currently call a separate service to know if a server is up or down. We should surface `monitor_status` directly on `ServerObject` so any server response tells the caller the current ON/OFF state in one call.

Key constraints discovered during exploration:
- `server-service` does **not** store server events — statuses are owned by `ontime-service`.
- Status must not be fetched in the handler layer (decided: service layer owns this).
- `ontime-service` already exposes `StatusService.GetCurrentStatuses(ctx, endpointIDs) -> map[endpointID]ServerStatus`, and `server-service` already has a `grpcclient.StatusClient` wired into `ServerRepository`.

## What Changes

- Add a `monitor_status` field (enum `ON` / `OFF`, nullable) to the `ServerObject` API schema and regenerate the ogen types.
- Extend `dto.Server` with a `MonitorStatus` field.
- `ServerService` (service layer) injects `StatusClient` and enriches every returned `dto.Server` with its current status by calling `GetCurrentStatuses` once per batch (List / Get / Search).
- `ToAPIServer` mapping sets `monitor_status` from `dto.Server.MonitorStatus`.
- Status fetch is best-effort: if the gRPC call fails, the service logs a warning and returns the server without a status (null) instead of failing the whole request.

## Capabilities

### New Capabilities
- `server-monitor-status`: Exposes the current monitoring status (ON/OFF) of a server on the server-service HTTP API `ServerObject` response, sourced from ontime-service's `StatusService`.

### Modified Capabilities
<!-- No existing spec-level requirements change; this is net-new behavior. -->

## Impact

- **API**: `ServerObject` gains `monitor_status` (nullable string enum). Breaking for strict clients that forbid unknown fields? No — additive field, non-breaking.
- **server-service code**:
  - `api/schemas/server.yaml` (schema) + regenerated `generated/api`.
  - `internal/dto/server.go` (new field).
  - `internal/service/server.go` (status enrichment).
  - `internal/handler/mapping.go` (map field).
  - `internal/app/injector.go` (`RegisterServerService` wires `StatusClient`).
- **Dependencies**: none new. Reuses existing `grpcclient.StatusClient` and `ontime-service` `StatusService`.
- **Cross-service**: none beyond what already exists; `ping-service`/`ontime-service` unchanged by this change.
