## Context

uptime-monitor is a microservice monorepo. Internal service-to-service communication uses gRPC (`common/proto` + generated packages `serverv1`, `eventv1`, ...). The HTTP layer (ogen-generated) is reserved for external/user-facing traffic via the API gateway, which sets `X-User-ID` from a JWT.

notification-service builds the daily digest Excel report. It needs two things from other services:
1. The list of a user's servers + their status counts — **already migrated to gRPC** in the prior `notification-grpc-server-count` change (works).
2. Per-server daily uptime statistics — still done over HTTP via `ontimeclient`, which posts to `POST /api/v1/servers/ontime/batch`. That endpoint does not exist (ontime only serves `GET /api/v1/servers/ontime` and `GET /api/v1/servers/ontime/{id}`), so the call 404s. The real `GET` endpoint is behind forward-auth (JWT), so it is unusable for internal calls anyway.

ontime-service computes daily uptime in `OntimeService.getServersOntime` → `Batcher.BatchGetOntime` (last-30-days stats per server, cached + DB-backed). It already serves gRPC on `:50052` (`eventv1.StatusService` with `GetCurrentStatuses`, `CountByStatus`) but offers no daily-ontime RPC. These stats are exactly what the digest needs.

## Goals / Non-Goals

**Goals:**
- Expose a gRPC RPC on ontime-service that returns per-server daily uptime stats for a user.
- Migrate notification-service's ontime client from the broken HTTP call to gRPC.
- Remove the dead HTTP ontime client code.
- Keep ontime-service's HTTP `GET /api/v1/servers/ontime` working for external/gateway clients.

**Non-Goals:**
- No JWT/auth interceptor on gRPC (internal calls pass `user_id` directly in the request, consistent with `StatusService` and `ServerService` methods).
- No change to the HTTP API contract or ogen-generated code for ontime.
- Not migrating notification-service's other HTTP clients (`userclient`→auth) — out of scope.

## Decisions

1. **New gRPC service `OntimeService` inside `event.v1`, not bolted onto `StatusService`.**
   `StatusService` means "current status" (online/offline, moment-in-time). Daily uptime history is a distinct concept, so a separate service keeps semantics clean. We keep it in the `event.v1` package because ontime-service already imports only `event.v1` for gRPC (avoids a 5th proto package). The new service is registered on the *same* `grpc.Server` instance already listening on `:50052`, so no new listener/port wiring is needed.
   - *Alternative:* add `GetServersOntime` to `StatusService` — rejected (semantic mismatch, as discussed).
   - *Alternative:* new proto package `ontime/v1` — rejected (more surface, ontime-service would import two proto packages for one server).

2. **RPC takes `user_id` only, returns all the user's servers' last-30-days stats.**
   The notification digest already fetches *all* of a user's servers (`serverclient.List` with `maxReportServers`), so a single `GetServersOntime(user_id)` call is sufficient and simplest. The `dates` filtering already happens downstream in `DigestService.buildReport` via `utils.TruncateDay`. This avoids a batch/per-item request shape.
   - *Alternative:* accept `(server_id, date)` pairs like the old HTTP body — rejected (unused filtering, more complex message, and ontime computes per-user anyway).

3. **Add `userID uint` to the `OntimeAdapter.GetServersOntimeForDates` interface.**
   The current signature `(ctx, servers, dates)` has no `userID`; the new gRPC call needs it. The only caller is `DigestService.SendReport`, which has `userID` in scope. Changing the interface (one caller) is cleaner than deriving `userID` from `servers[0]` (fragile when `servers` is empty).

4. **notification-service gets a dedicated gRPC dial config for ontime, mirroring ping-service.**
   ping-service already patterns this: `GRPCConfig` has `ServerAddr` (server-service) and `EventAddr` (ontime gRPC `:50052`), each with its own `GRPCClientWrapper`. notification-service reuses the same shape — add `EventAddr` to its `GRPCConfig` (default `ontime-service:50052`) plus a second `GRPCClientWrapper`/`RegisterGRPCOntimeClient`. This keeps the existing server-service gRPC path untouched.

5. **Remove the HTTP ontime client entirely** (no remaining caller after migration).

## Risks / Trade-offs

- [Risk] gRPC target misconfiguration (wrong port/host) → digest still fails, but now with a clear connection error logged via the new client logs. → Mitigation: default `ontime-service:50052` matches ontime's actual gRPC port; error logged at error level with target+user.
- [Risk] `GetServersOntime` depends on the ontime DB/cache; if down, stats return empty but report still generates. → Non-breaking; same as current degraded behavior.
- [Trade-off] Adding a public `GetServersOntime` method to `OntimeService` exposes what was private `getServersOntime` — acceptable, it's the intended public boundary.
- [Trade-off] A new proto RPC is backward-compatible for existing gRPC callers (ping-service uses `EventRecorderService`/`StatusService`) — they ignore the new method.

## Migration Plan

1. Edit proto, run `buf generate` in `common/proto`.
2. Add gRPC handler + public service method in ontime-service; register service in `app/server.go`; build ontime-service.
3. Add gRPC client wiring in notification-service; rewrite `ontimeclient` to gRPC; update `config.go`, `viper.go`, `config.yml`, `injector.go`, `OntimeAdapter` interface; build notification-service.
4. Build and test both services.
5. Deploy ontime-service first (adds RPC), then notification-service (switches transport). Rollback: revert notification-service to HTTP client (git revert) — but HTTP endpoint is broken anyway, so rollback is only meaningful for ontime if the new RPC causes issues.

## Open Questions

- None blocking.
