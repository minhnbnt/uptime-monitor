## Why

Iteration 1 updated server-service to store k8s identity fields and changed all gRPC proto contracts, but ping-service and ontime-service still reference the old HTTP-based fields (url, method, expected_code) and endpoint_id. The code doesn't compile. Iteration 2 completes the migration: ping-service replaces HTTP pinging with k8s pod status checking, and ontime-service aligns its data model with server_id.

## What Changes

- **BREAKING**: ping-service domain model changes from HTTP endpoint (url, method, expected_code, body_check_expr) to k8s identity (namespace, kind, object_id, container_name)
- **BREAKING**: ping-service replaces HTTP client with k8s client-go to query pod/container status
- **BREAKING**: ping-service ZSet scheduler stores k8s identity fields instead of HTTP fields
- **BREAKING**: ping-service Debezium CDC consumer listens to `servers` topic instead of `endpoints`
- **BREAKING**: ontime-service `server_events` table changes `endpoint_id` → `server_id`
- **BREAKING**: ontime-service gRPC proto contracts change `endpoint_id` → `server_id`
- Remove HTTP-specific infrastructure: `responseChecker.go`, `bodychecker.go`
- Add k8s client infrastructure: client-go, in-cluster config, RBAC ServiceAccount
- ping-service `Ping` gRPC server accepts k8s fields and returns running (bool)

## Capabilities

### New Capabilities
- `k8s-pod-status-checker`: K8s client infrastructure that queries pod phase and container readiness via client-go, replacing HTTP ping

### Modified Capabilities

## Impact

- **ping-service domain**: `domain/endpoint.go` — replace HTTP fields with k8s identity fields
- **ping-service infrastructure**: delete `ping.go` (HTTP client), `bodychecker.go`; add `k8sclient/` package
- **ping-service service**: rewrite `pingloop.go` (k8s check instead of HTTP), delete `responsechecker.go`
- **ping-service scheduler**: update `endpointcache.go`, `endpointprovider.go` field mapping
- **ping-service gRPC server**: update `ping_server.go` to accept k8s fields
- **ping-service gRPC client**: update `endpoint_client.go` field mapping
- **ping-service CDC**: update `streamevents.go` topic, `processor.go` parsing
- **ontime-service proto**: `event_service.proto` — `endpoint_id` → `server_id`
- **ontime-service domain**: `serverevent.go` — `EndpointID` → `ServerID`
- **ontime-service DB migration**: `server_events` table — rename column
- **ontime-service handlers**: update all gRPC handlers for new field names
- **Dependencies**: add `k8s.io/client-go` to ping-service
- **K8s RBAC**: ServiceAccount with pods/get, deployments/get, statefulsets/get, daemonsets/get, replicasets/get permissions
