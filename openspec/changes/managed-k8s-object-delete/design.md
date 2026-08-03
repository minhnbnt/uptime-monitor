## Context

See proposal.md — Why/What for motivation. Current state:

- `server-service` has `POST /api/v1/k8s-objects` (creates pod + server record) and an unsecured `DELETE /api/v1/k8s-objects?namespace&object_id` (no ownership check, guarded only by a 409 when a server still references the object).
- Server delete is a GORM soft delete (`deleted_at`); normal GORM scopes exclude soft-deleted rows.
- `domain.Server` embeds `gorm.Model`; migrations run via `AutoMigrate`.
- UI (`uptime-monitor-ui`) has `ServerObject` with `namespace/kind/object_id`, `apiDeleteServer`, and a plain `window.confirm` delete on `ServerDetail`. It has no toast system (logout uses a modal dialog).

## Goals / Non-Goals

**Goals:**
- Gate pod deletion behind ownership + a new `managed` flag.
- Keep the "delete server, then delete pod" flow valid despite server soft-delete.
- Expose `managed` so the UI can decide exactly when to offer pod deletion.
- Minimal, reviewable change touching server-service + UI.

**Non-Goals:**
- No new endpoint for listing untracked objects.
- No hard-delete of server records.
- No multi-user role model beyond ownership (`created_by_id == userID`).

## Decisions

### 1. `managed` is a server-scoped boolean (default false)
`domain.Server.Managed bool` (`gorm:"type:boolean;not null;default:false"`); `CreateK8sObject` sets `true`. Manual create/import leave it false.
- Alternative: track provenance in a separate table — overkill; a single flag matches the current create/register dichotomy.
- Rationale: it is the source of truth for "may this system delete the pod".

### 2. Delete authorization via an Unscoped server lookup
`DeleteK8sObject(ctx, userID, ns, objID)`:
1. `GetByNamespaceObjectIDUnscoped(ns, objID)` → `*domain.Server` (includes soft-deleted).
2. none → `ErrNotFound` (404).
3. `CreatedByID != userID` → `ErrForbidden` (403).
4. `!Managed` → `ErrForbidden` (403).
5. else → `DeletePod` → 204.
- Uses the soft-delete marker (the row persists with `deleted_at`) so ownership/managed survive a prior server delete — exactly the "delete pod after deleting server" flow.
- Replaces the old 409 (`ErrPodMonitored`) guard: now managed+owned pods may be deleted even while the server is active (per user decision).
- Alternative: two-step UI-only sequencing relying on the old guard — rejected because it doesn't fix the ownership hole.

### 3. Handler passes the authenticated user
`DeleteK8sObject` handler passes `authclient.GetUserID(ctx)` into the service.

### 4. `managed` exposed on the server API
Add `managed` to the server OpenAPI schema + `dto.Server`, regenerate with `ogen` (`go generate ./...`). UI `ServerObject` adds `managed?: boolean`.
- Needed so the UI can gate the suggestion and the future Delete-Pod control.

### 5. UI suggestion flow (delete server, then pod)
On `ServerDetail`, before deleting capture the server's `namespace`/`object_id`/`managed`/`kind`. After a successful `DELETE /servers/{id}`:
- If `managed && kind === 'Pod'` → show "Delete Pod `<ns>/<obj>` too?" prompt; on accept call `apiDeleteK8sObject(ns, obj)`, else dismiss.
- Ordering is server-then-pod; with decision #2 the pod call is authorized by ownership + managed (server already soft-deleted, found via Unscoped).
- Because the repo has no toast system, render a lightweight prompt — a fixed suggestion component rendered in `Layout` (context-driven) OR an inline panel before navigation. Chosen at implementation (A/B), defaulting to a small inline panel on `ServerDetail` to avoid new global infra.

## Risks / Trade-offs

- **Deleting a managed pod while its server is still active leaves a dangling reference** → Accepted by decision (managed = we own it); server continues to monitor a now-missing pod and will surface OFF.
- **Unscoped lookup must exist** → Add `GetByNamespaceObjectIDUnscoped`; ensure it maps DB errors to `ErrInternal` and empty to `ErrNotFound`.
- **`managed` defaults** → false for all existing rows (safe; existing manual servers stay non-deletable until explicitly set).
- **UI ordering** → Must await `DELETE /servers/{id}` before `DELETE /k8s-objects` to avoid coupling on any leftover guard; sequential `await`.

## Migration Plan

1. Deploy server-service with the new `managed` column (`AutoMigrate` adds it; existing rows default `false`).
2. Update and deploy UI.
3. Redeploy endpoints; no data backfill required beyond the column default.

Rollback: revert the chart/image. No destructive data changes.

## Open Questions

- Where to surface the UI prompt (global toast in `Layout` vs inline panel) — a presentation choice that does not change specs/approach; settled during implementation.
