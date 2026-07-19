## Context

The uptime-monitor system is a microservice monorepo. Internal service-to-service communication uses gRPC (`common/proto` + generated `serverv1`, `eventv1`, ...). The HTTP layer (ogen-generated) is reserved for external/user-facing traffic via the API gateway, which sets `X-User-ID` from a JWT.

Today, server-service exposes both an HTTP API and a gRPC `ServerService`. importer-service and ontime-service already call server-service over gRPC. notification-service is the exception: it calls server-service over HTTP (`serverclient.Client`) for `List` and `CountByStatus`. This mismatch was the root cause of the recent digest outage — the HTTP route `/api/v1/servers/count` did not exist, producing 404s that collapsed into a generic `ErrInternal` and triggered infinite Temporal retries.

The repository already has `ServerService.CountByStatus(ctx, userID)` (service/repo layer) and an ogen HTTP `GET /api/v1/servers/count` endpoint. The missing piece is a gRPC RPC so internal callers can reach the same logic without HTTP.

## Goals / Non-Goals

**Goals:**
- Add a gRPC `CountServersByStatus` RPC to `server.v1.ServerService`.
- Migrate notification-service's server-service client from HTTP to gRPC for both `List` and `CountByStatus`.
- Remove the dead HTTP server client from notification-service.
- Keep the HTTP `/api/v1/servers/count` endpoint working for external clients.

**Non-Goals:**
- No JWT/auth interceptor on gRPC (internal calls pass `user_id` directly in the request, consistent with existing `ListServersRequest`/`GetServerRequest`).
- No change to the HTTP API contract or ogen-generated code for `/api/v1/servers/count`.
- Not migrating notification-service's other HTTP clients (userclient→auth, ontimeclient→ontime) — out of scope.

## Decisions

1. **Reuse existing service method, add thin gRPC handler.** `ServerService.CountByStatus` already exists; the gRPC handler simply maps the proto request/response. No new business logic.
   - *Alternative:* Implement count logic again in the handler — rejected (duplication).

2. **Pass `user_id` in the gRPC request message** (`CountServersByStatusRequest.user_id`), mirroring `ListServersRequest`. This matches every existing gRPC method in this service and avoids building a JWT metadata interceptor that notification-service (an internal worker) has no token for anyway.

3. **Notification-service gets a dedicated `grpc.server_addr` config** (plain `host:port`, e.g. `localhost:50051`), copied from the importer/ontime pattern (`GRPCClientWrapper` + `RegisterGRPCClient`). The existing `server_service.addr` field carries a `grpc://` scheme prefix which `grpc.NewClient` cannot parse as a target, so a separate plain-address field is used instead of reusing it.

4. **Remove the HTTP `serverclient` package entirely** rather than leaving it as dead code, since no caller remains after the migration.

5. **Keep HTTP `/api/v1/servers/count`** as-is — it is the gateway-facing contract and external clients depend on it.

## Risks / Trade-offs

- [Risk] gRPC target misconfiguration (wrong port) → digest still fails, but now with a clear connection error logged via the new client logs. → Mitigation: default `grpc.server_addr: localhost:50051` matches server-service's actual gRPC port (`50051`).
- [Risk] `CountByStatus` repo depends on event-service (gRPC) for online/offline; if that link is down, counts return 0/0. → Non-breaking; report still generates. No change to this behavior.
- [Trade-off] A new proto RPC is backward-compatible for existing gRPC callers (importer, ontime) — they simply ignore the new method.

## Migration Plan

1. Edit proto, run `buf generate` in `common/proto`.
2. Add gRPC handler in server-service; build server-service.
3. Add `config/grpc.go` + gRPC client in notification-service; rewrite `serverclient` to gRPC; wire injector + config.
4. Build and test both services.
5. Deploy server-service first (adds RPC), then notification-service (switches transport). Rollback: revert notification-service to HTTP client (git revert) — HTTP endpoint remains available throughout.

## Open Questions

- None blocking. Optional future: migrate notification-service's `List` callers consistently (already done in this change) and consider migrating userclient/ontimeclient to gRPC for full consistency.
