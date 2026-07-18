## Context

`server-service` owns server/endpoint CRUD but does **not** store monitoring events. Uptime events are recorded and read by `ontime-service`, which now exposes two gRPC services: `EventRecorderService` (write) and `StatusService` (read: `GetCurrentStatuses`, `CountByStatus`). `server-service` already has a `grpcclient.StatusClient` (wrapping `StatusService`) injected into `ServerRepository` and used by `CountByStatus`.

Today `ServerObject` carries no live status, so API consumers cannot tell if a server is ON/OFF from a server response. This design adds `monitor_status` to `ServerObject`, sourced from `StatusService.GetCurrentStatuses`, enriched in the service layer (not the handler).

Server↔endpoint is 1-1 (`endpoints.server_id`). `GetCurrentStatuses` keys by `endpointID`, so enrichment maps `server.Endpoint.ID` → status.

## Goals / Non-Goals

**Goals:**
- Expose current `monitor_status` (ON/OFF) on every `ServerObject` returned by List / Get / Search.
- Fetch status in the service layer via a single batched gRPC call per request.
- Degrade gracefully: status is best-effort; a status fetch failure must not fail the server request.

**Non-Goals:**
- Storing events in `server-service` (explicitly out of scope).
- Changing the `StatusService` proto or `ontime-service` behavior.
- Adding status to `CreateServer`/`UpdateServer` request/response shapes beyond the read path.
- Real-time/pushed status; this is a point-in-time read.

## Decisions

1. **Enrich in service layer, not handler.**
   - Rationale: handler stays a thin boundary (map DTO→API). Status is a domain attribute of a server, so `ServerService` owns fetching it. Handler fetch (alternative A) was rejected by the user.
   - Alternative considered: handler calls `StatusClient` then passes into mapping — rejected to keep handlers dumb and avoid giving handlers cross-service clients.

2. **Add `MonitorStatus` to `dto.Server` (not a separate response struct).**
   - Rationale: status is a property of the server entity; carrying it on `dto.Server` lets List/Get/Search all reuse the same mapping. `ServerFromDomain` leaves it zero-value (status never comes from DB).

3. **One batched `GetCurrentStatuses` call per request.**
   - Collect `endpointID`s from the returned servers (skip servers without an endpoint), call once, then map results back. Avoids N+1.

4. **`monitor_status` schema field is nullable / optional.**
   - Rationale: a freshly created server may have no events yet → status unknown → `null`. ogen generates `OptString` for an optional string; map with `api.NewOptString(string(status))` (empty → null).

5. **Best-effort on gRPC error.**
   - On `GetCurrentStatuses` error: `logger.Warn`, return servers without status (empty string → null in API). The request still succeeds.

## Risks / Trade-offs

- [Risk] `ontime-service` down or slow → every server list/get adds latency / may return null statuses. → Mitigation: best-effort (warn + continue); call is a single batched request, not per-server.
- [Risk] Search endpoint latency increases by one gRPC round-trip. → Mitigation: still a single batched `GetCurrentStatuses`; acceptable. If needed later, search can skip enrichment (not done now for consistency).
- [Risk] `endpointID` vs `serverID` confusion (proto keys by endpoint). → Mitigation: explicitly map via `server.Endpoint.ID`, with a nil-endpoint guard.

## Migration Plan

- Schema + code change is additive (new nullable field) → no breaking API change, no DB migration.
- Regenerate ogen types (`go generate ./...` in server-service).
- Rollback: revert the change; field disappears from responses. No data migration needed.

## Open Questions

- None outstanding. Search enrichment included by default for consistency.
