## Purpose

Defines the backend contract for managing the lifecycle of Kubernetes objects (pods) that the system creates, including the `managed` flag, ownership enforcement, and authorized deletion through the k8s-objects API — including objects whose server record has already been soft-deleted.

## ADDED Requirements

### Requirement: Servers carry a managed flag
Every server SHALL indicate whether its backing Kubernetes object is managed by the system. A server created via `POST /k8s-objects` SHALL have `managed = true`; a server registered manually (`POST /servers` or import) SHALL have `managed = false`.

#### Scenario: System-created object is managed
- **WHEN** a server is created through `POST /api/v1/k8s-objects`
- **THEN** the returned server has `managed = true`

#### Scenario: Manually registered object is not managed
- **WHEN** a server is registered through `POST /api/v1/servers`
- **THEN** the returned server has `managed = false`

### Requirement: Deleting a k8s object requires ownership
The system SHALL reject `DELETE /api/v1/k8s-objects` with HTTP 403 when the requesting user did not create the server referencing the object.

#### Scenario: owner deletes own object
- **WHEN** the owner calls `DELETE /api/v1/k8s-objects?namespace=…&object_id=…`
- **THEN** the pod is deleted and the response is HTTP 204

#### Scenario: non-owner is rejected
- **WHEN** a user who is not the server's owner calls `DELETE /api/v1/k8s-objects` for that object
- **THEN** the system responds HTTP 403 and the pod is not deleted

### Requirement: Deleting a k8s object requires it to be managed
The system SHALL reject `DELETE /api/v1/k8s-objects` with HTTP 403 when the server referencing the object has `managed = false` (an externally registered workload).

#### Scenario: managed object is deletable
- **WHEN** a user calls `DELETE /api/v1/k8s-objects` for a managed object they own
- **THEN** the pod is deleted with HTTP 204

#### Scenario: unmanaged object is not deletable
- **WHEN** a user calls `DELETE /api/v1/k8s-objects` for an object whose server has `managed = false`
- **THEN** the system responds HTTP 403 and the pod is not deleted

### Requirement: Object remains deletable after its server is soft-deleted
The ownership and `managed` checks SHALL consider the server record even when it has been soft-deleted, so a pod can be deleted after its server was deleted.

#### Scenario: delete pod after deleting server
- **WHEN** a server (managed, owned) is deleted (soft-deleted) and then its owner calls `DELETE /api/v1/k8s-objects` for the pod
- **THEN** the checks pass and the pod is deleted with HTTP 204

#### Scenario: no matching server
- **WHEN** `DELETE /api/v1/k8s-objects` matches no server record (active or soft-deleted) for the namespace/object_id
- **THEN** the system responds HTTP 404
