## Purpose

Defines the frontend behavior that, when a user deletes a server which backs a managed Kubernetes pod, offers to also delete that pod through the k8s-objects API so the underlying workload is cleaned up.

## ADDED Requirements

### Requirement: Suggest deleting the pod when deleting a managed server
When a user deletes a server whose `managed` is true (a pod the system created), the application SHALL, after the server is deleted, offer the user the option to also delete the pod.

#### Scenario: delete a managed server offers pod deletion
- **WHEN** the user deletes a server with `managed = true` and `kind = Pod`
- **THEN** the server is deleted first, and then the user is shown a prompt to also delete the pod

#### Scenario: deleting a non-managed server offers no pod deletion
- **WHEN** the user deletes a server with `managed = false`
- **THEN** the pod-deletion prompt is not shown

### Requirement: Pod deletion follows server deletion
When the user chooses to delete the pod, the application SHALL delete the server record before calling `DELETE /api/v1/k8s-objects` for that pod.

#### Scenario: accepting the suggestion deletes server then pod
- **WHEN** the user accepts the "delete pod too" suggestion for a managed server
- **THEN** `DELETE /api/v1/servers/{id}` is called before `DELETE /api/v1/k8s-objects?namespace=…&object_id=…`

#### Scenario: dismissing the suggestion keeps the pod
- **WHEN** the user dismisses the pod-deletion prompt
- **THEN** only the server is deleted and the pod remains in the cluster
