## 1. Backend — managed flag & data model

- [x] 1.1 Add `Managed bool` field to `domain.Server` in server-service (gorm `not null;default:false`)
- [x] 1.2 Set `Managed: true` in `K8sObjectService.CreateK8sObject` before creating the server
- [x] 1.3 Add `GetByNamespaceObjectIDUnscoped(ctx, namespace, objectID)` to `ServerRepository` (includes soft-deleted rows via `Unscoped`)

## 2. Backend — secure k8s-object delete

- [x] 2.1 Change `K8sObjectService.DeleteK8sObject` signature to accept `userID uuid.UUID`
- [x] 2.2 Implement ownership + `managed` gating: 404 when no server matches, 403 for non-owner, 403 for `managed == false`, else delete the pod
- [x] 2.3 Update `K8sObjectHandler.DeleteK8sObject` to pass `authclient.GetUserID(ctx)`
- [x] 2.4 Remove the old `ExistsByNamespaceObjectID` / `ErrPodMonitored` 409 check (replaced by the new gating)

## 3. Backend — expose `managed` on the API

- [x] 3.1 Add `managed` to the server OpenAPI schema and `dto.Server`
- [x] 3.2 Regenerate ogen code (`go generate ./...`) and confirm server-service builds + lints

## 4. Frontend — delete pod API + suggestion

- [x] 4.1 Add `apiDeleteK8sObject(namespace, objectID)` in `src/lib/api.ts` (`DELETE /api/v1/k8s-objects`)
- [x] 4.2 Add `useDeleteK8sObject()` in `src/lib/queries.ts`
- [x] 4.3 Add `managed?: boolean` to `ServerObject` in `src/types/api.ts`
- [x] 4.4 In `ServerDetail.tsx`, capture `managed`/`namespace`/`object_id` before delete; after a successful server delete and only when `managed && kind === 'Pod'`, show a "Delete Pod too?" prompt
- [x] 4.5 On accept, call `apiDeleteK8sObject` after the server delete; on dismiss, leave the pod in place
- [x] 4.6 Run `pnpm` typecheck/lint to validate UI changes

## 5. Validation

- [x] 5.1 Run server-service `go build ./...`, `go vet`, and `golangci-lint`
- [ ] 5.2 Exercise the delete-pod path (owner/managed success, 403 non-owner, 403 unmanaged, 404 missing) via tests or manual API
