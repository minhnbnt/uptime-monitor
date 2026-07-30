## Why

The system currently only supports checking K8s pod/container status (pod phase + container readiness) for monitoring endpoints. Users need the ability to perform HTTP health checks through K8s DNS resolution for Service, Pod, and StatefulSet resources — e.g., calling `http://<service>.<namespace>.svc.cluster.local:<port>/health`.

## What Changes

- Add new Kind `Service` to the kind enum
- Add new table `server_http_configs` (server_id PK/FK, port, endpoint_path) — existence of a record indicates http-dns mode
- Add proto enum `PingType` and optional message `HttpDnsConfig` to `EndpointData` and `PingRequest`
- Add optional `http_config` object to server-service API request/response schemas
- When `PingType = HTTP_DNS`, resolve K8s DNS + HTTP GET; when `PingType = POD_STATUS`, use existing pod-status check
- Propagate through: gRPC, API, ping-service execution, CDC, Redis cache

## Capabilities

### New Capabilities
- `dns-http-ping`: Ability to configure endpoints with HTTP health checks targeting DNS-resolved K8s resources (Service, Pod, StatefulSet) via a new `server_http_configs` table and `PingType` proto enum

### Modified Capabilities

- (none — no existing specs)

## Impact

- **Proto**: `endpoint_service.proto` and `ping_service.proto` — new enum `PingType`, new message `HttpDnsConfig`, new optional fields on existing messages
- **Server-service API**: OpenAPI spec (kind enum, optional `http_config` object), handler changes, gRPC endpoint server changes
- **Ping-service**: k8s client gets DNS + HTTP health check logic; ping loop dispatches by `PingType` enum; gRPC ping server forwards enum + config
- **CDC / Cache**: CDC processor, endpoint cache, gRPC endpoint client — new field mapping (config objects, not flat columns)
- **DB**: New table `server_http_configs` via GORM AutoMigrate
