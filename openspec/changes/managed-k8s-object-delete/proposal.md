## Why

The `DELETE /api/v1/k8s-objects` endpoint can delete any pod by namespace/object_id with no ownership or provenance check, so any authenticated user can delete someone else's pod. Server deletion is only a soft delete (`deleted_at`), leaving the underlying pod in the cluster with no way for the owner to clean it up safely.

## What Changes

- **Add a `managed` flag to servers** so a pod that the system created (`POST /k8s-objects`) is distinguished from externally registered workloads. Only managed objects may be deleted through the API.
- **Secure `DELETE /api/v1/k8s-objects`**: require the caller to be the owner (`created_by_id == userID`) and require the server to be `managed`. Reject otherwise (`403`); reject `404` when no server matches.
- **Read soft-deleted servers (Unscoped)** in the delete check so an object stays deletable after its server was soft-deleted (the "delete server, then delete pod" flow).
- **Expose `managed` in the server API/DTO** so the UI can decide whether to offer pod deletion.
- **UI: suggest deleting the pod when deleting a server** — after a server that is a managed pod is deleted, prompt the user to also delete the pod via the API (delete server first, then delete pod, to keep ordering valid).
- Remove the old 409 `ErrPodMonitored` guard, replaced by the ownership + `managed` checks.

## Capabilities

### New Capabilities
- `k8s-object-management`: backend contract for managing the lifecycle of k8s objects (pods) created by the system — a `managed` flag, owner enforcement, and authorized deletion through `DELETE /k8s-objects` (including after the server record is soft-deleted).
- `pod-delete-suggestion`: UI behavior that, when a user deletes a server backing a managed pod, offers to also delete that pod via the API.

### Modified Capabilities
<!-- None: openspec/specs is empty; no existing spec requirements change. -->
- (none)

## Impact

- **server-service**: `domain.Server` (new `Managed` field + AutoMigrate), `K8sObjectService.DeleteK8sObject` (ownership + `managed` gating, `Unscoped` lookup), repository lookup method, handler passes `userID`, DTO + API schema expose `managed`, ogen regeneration.
- **uptime-monitor-ui**: `apiDeleteK8sObject`, `useDeleteK8sObject`, `ServerObject.managed`, and a "delete pod too?" suggestion on `ServerDetail`.
- No new external dependencies.
