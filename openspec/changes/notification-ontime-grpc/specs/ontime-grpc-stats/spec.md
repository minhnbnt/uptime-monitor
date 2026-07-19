## ADDED Requirements

### Requirement: ontime-service exposes daily uptime over gRPC
ontime-service SHALL expose a gRPC service `OntimeService` (in `event.v1`) with an RPC `GetServersOntime` that accepts a `user_id` and returns, for each of that user's servers, the daily uptime statistics (`date` and `stats`) for the last 30 days.

#### Scenario: Successful retrieval for a user with servers
- **WHEN** a gRPC caller invokes `GetServersOntime` with a valid `user_id`
- **THEN** the server returns a list of per-server entries, each containing `server_id` and a list of `ontime_stats` (`date`, `stats`) for the last 30 days

#### Scenario: User with no servers
- **WHEN** a gRPC caller invokes `GetServersOntime` for a `user_id` that owns no servers
- **THEN** the server returns an empty list of server entries without error

### Requirement: gRPC daily uptime reuses existing computation
The gRPC `GetServersOntime` RPC SHALL return the same daily uptime statistics that the existing HTTP `GET /api/v1/servers/ontime` endpoint returns for the same user.

#### Scenario: Parity between gRPC and HTTP
- **WHEN** the same `user_id` is queried via gRPC `GetServersOntime` and via HTTP `GET /api/v1/servers/ontime`
- **THEN** both responses contain identical per-server daily `stats` values

### Requirement: HTTP endpoint remains available
ontime-service SHALL continue to serve `GET /api/v1/servers/ontime` over HTTP for external/gateway clients.

#### Scenario: External HTTP client still works
- **WHEN** an external client calls `GET /api/v1/servers/ontime` with a valid JWT
- **THEN** the server returns the same per-server daily uptime JSON as before
