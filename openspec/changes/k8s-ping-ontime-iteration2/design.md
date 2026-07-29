## Context

After iteration 1, server-service stores k8s identity fields (namespace, kind, object_id, container_name) and all gRPC proto contracts use these fields. However, ping-service and ontime-service still reference the old HTTP-based fields and endpoint_id. The system doesn't compile.

ping-service currently:
- Has an HTTP client (`PingClient`) that makes GET/POST/etc. requests to URLs
- Uses a `ResponseChecker` to validate HTTP status codes and body expressions
- Schedules periodic pings via Redis ZSet
- Consumes Debezium CDC from `uptime.public.endpoints` stream
- Records status via gRPC to ontime-service using `endpoint_id`

ontime-service currently:
- Stores `server_events` with `endpoint_id` column
- Serves `GetCurrentStatuses(endpoint_ids)` and `CountByStatus(user_id)`
- Proto contracts use `endpoint_id` everywhere

## Goals / Non-Goals

**Goals:**
- Replace HTTP pinging with k8s pod/container status checking via client-go
- Update ping-service domain model from HTTP endpoint to k8s identity
- Update ontime-service to track status by server_id instead of endpoint_id
- Update Debezium CDC consumer to listen to `servers` topic
- Maintain the existing ZSet scheduler architecture (proven, sharded, handles thundering herd)

**Non-Goals:**
- k8s resource lifecycle management (apply/delete) — still manual via kubectl
- Support for multi-cluster k8s — single cluster assumed
- Custom health check endpoints on pods — only k8s-native readiness
- Dashboard or UI changes — backend only
- Data migration from existing production data (pre-production system)

## Decisions

### D1: Pod status check logic

**Choice**: Query pods directly via client-go. For Pod kind: `GET namespaced pod`. For Deployment/StatefulSet/DaemonSet/ReplicaSet: `LIST pods with label selector`. A pod is "running" if `pod.phase == Running` AND all container statuses have `ready == true`. If `container_name` is specified, only check that container.

**Alternatives considered**:
- Use readiness probe result only → rejected: requires pods to expose probes, not always available
- Use deployment.status.availableReplicas > 0 → rejected: doesn't work for Pod kind, and doesn't catch individual container failures
- Use custom health check endpoint → rejected: adds complexity, defeats purpose of native k8s monitoring

**Rationale**: Direct pod query is the most reliable indicator of "is this thing actually running?". Checking both phase and container readiness catches partial failures (e.g., init container stuck, sidecar crash-looping).

### D2: K8s client infrastructure

**Choice**: Use `k8s.io/client-go` with in-cluster config. Create a `K8sClient` interface for testability. RBAC ServiceAccount with `pods/get` and `pods/list` permissions.

**Alternatives considered**:
- Dynamic client → rejected: typed client is simpler and sufficient for pod queries
- Out-of-cluster config → rejected: service runs in-cluster, in-cluster is natural
- REST client directly → rejected: client-go provides typed client with less boilerplate

**Rationale**: client-go is the standard. In-cluster config works automatically with ServiceAccount. Interface allows mocking for tests.

### D3: Domain model rename

**Choice**: In ping-service, rename `Endpoint` to `Server` with k8s identity fields (namespace, kind, object_id, container_name, interval, timeout). Remove URL, Method, ExpectedCode, BodyCheckExpr.

**Rationale**: Aligns with server-service's unified model. The "endpoint" concept no longer exists — the server IS the thing being monitored.

### D4: Status tracking by server_id

**Choice**: ontime-service `server_events` table changes `endpoint_id` → `server_id`. All gRPC contracts change accordingly. ping-service sends `server_id` when recording events.

**Rationale**: Since there's no separate endpoint, the server is the entity being monitored. This is a clean mapping from iteration 1's design decision D3.

### D5: Debezium CDC stream topic

**Choice**: Change stream key from `uptime.public.endpoints` to `uptime.public.servers`. Update Debezium config to watch `public.servers` only.

**Rationale**: The endpoints table no longer exists. Debezium already watches `public.servers` (updated in iteration 1). ping-service needs to consume from the new topic.

## Risks / Trade-offs

- **[k8s API rate limiting]** → If many servers are monitored, frequent pod queries could hit API server limits. Mitigation: use Informer/cache instead of direct GET/LIST for high-volume scenarios. For iteration 1 scale (< 1000 servers), direct queries are fine.

- **[RBAC complexity]** → ServiceAccount needs proper ClusterRole/RoleBinding. Mitigation: provide RBAC YAML manifests in the change.

- **[Proto breaking changes]** → All services must be deployed together. Mitigation: pre-production system, coordinated deploy acceptable.

- **[No multi-cluster support]** → Single cluster assumption. Mitigation: acceptable for iteration 2; can add cluster selector field later.

- **[Container name ambiguity]** → If container_name is empty and pod has multiple containers, which one to check? Mitigation: check all containers; return running only if ALL are ready.
