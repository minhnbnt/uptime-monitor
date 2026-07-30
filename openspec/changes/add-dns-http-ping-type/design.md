## Context

The current system supports only pod-status checks (Pod phase + container readiness). See `proposal.md – Why` for motivation. The data flow for a ping is:

```
server-service (stores Server config) 
  → gRPC EndpointData 
  → ping-service 
    → k8s client CheckPodStatus 
    → event-service (status record)
```

This design extends the data model and ping execution to support a second ping type without breaking the existing behavior.

## Goals / Non-Goals

**Goals:**
- Add new table `server_http_configs` (1-1 with servers) for http-dns configuration
- Implement DNS-based HTTP health check logic in ping-service k8s client
- Support kind `Service` (new) and existing `Pod`/`StatefulSet` for http-dns mode
- Propagate through gRPC (enum `PingType` + optional message `HttpDnsConfig`), API (optional `http_config` object), CDC, Redis cache
- Zero migration cost for existing records — no existing row has a `server_http_configs` record, so all fall back to pod-status behavior

**Non-Goals:**
- No changes to event-service
- No changes to the frontend/UI (API changes only)
- No new protobuf services or RPCs — only enum, message, and field additions to existing messages

## Decisions

### 1. Separate table instead of columns on servers
- **Decision**: HTTP-DNS config lives in a separate table `server_http_configs` (server_id PK/FK → servers.id ON DELETE CASCADE) instead of adding `ping_type`, `port`, `endpoint_path` columns to `servers`
- **Rationale**: The config is optional and only relevant for http-dns mode. A separate table avoids nullable columns on the main table and makes the intent explicit — existence of a record = http-dns mode. No `ping_type` column needed anywhere in the DB.
- **Alternative considered**: Adding columns to `servers` with nullable/default — rejected per user preference for normalized optional config

### 2. Proto enum PingType + optional HttpDnsConfig message
- **Decision**: Define an enum `PingType { PING_TYPE_POD_STATUS = 0; PING_TYPE_HTTP_DNS = 1; }` and a message `HttpDnsConfig { int32 port; string endpoint_path; }`. Both `EndpointData` and `PingRequest` gain `PingType ping_type` and `optional HttpDnsConfig http_dns_config`.
- **Rationale**: The enum makes dispatch explicit and type-safe at the proto level. The separate message groups port + path cleanly. `optional` in proto3 correctly distinguishes unset from zero. Breakage radius = zero (new fields, old clients ignore them).
- **Alternative considered**: Flat `ping_type` string field — rejected in favor of type-safe enum. Flat `port`/`endpoint_path` — rejected in favor of grouped message. Inferring PingType from config presence — rejected per user preference for explicit dispatch at proto level.

### 3. DNS resolution strategy
- **Decision**: Use `<name>.<namespace>.svc.cluster.local` for Service, retrieve Pod IP directly via K8s API for Pod, use `<name>-0.<name>.<namespace>.svc.cluster.local` for StatefulSet
- **Rationale**: Standard K8s DNS naming. Service DNS is always available. For Pods, the IP-based DNS format (`<ip-dashes>.<namespace>.pod.cluster.local`) is fragile and depends on cluster DNS configuration — fetching the IP directly from the API is more reliable. For StatefulSet, assuming pod-0 and the headless service name matching the StatefulSet name covers the common case.
- **Alternative considered**: Using ClusterIP from Service object instead of DNS — rejected in favor of proper DNS resolution as requested.

### 4. HTTP client for health checks
- **Decision**: Use `net/http` directly in the k8s client package for the HTTP GET call, respecting context timeout
- **Rationale**: A single GET request needs no client customization beyond timeout. The existing `infrastructure.PingClient` is unused dead code and not worth wiring up. Using `http.DefaultClient` with context cancellation is the simplest correct approach.
- **Alternative considered**: Reusing `infrastructure.PingClient` — rejected because it would require dependency injection into the k8s client, adding coupling without benefit.

### 5. API http_config nested object
- **Decision**: Server create/update requests accept an optional `http_config` object: `{port, endpoint_path}`. Server responses include `http_config` as a nullable object. Flat fields are not added to the server schema.
- **Rationale**: Mirrors the DB normalization. Cleaner API surface — http-dns config is a cohesive unit, not three independent fields.

## Risks / Trade-offs

- **[Dual write]** Creating/updating a server with `http_config` touches two tables — risk of partial failure
  → Mitigation: Wrap in a single transaction (GORM transaction in service layer)
- **[StatefulSet DNS assumption]** Hardcoding pod-0 is fragile for multi-replica StatefulSets with partitioned updates
  → Mitigation: This covers the common single-replica case. Future enhancement could add a `pod_index` or `target_pods` field.
- **[CDC partial data]** The debezium CDC stream only emits the `servers` table row, not the joined `server_http_configs` row
  → Mitigation: ping-service always fetches full `EndpointData` via gRPC GetEndpoints on cache miss. The CDC event is only used to trigger rescheduling (register/unregister). The real data comes from gRPC.
