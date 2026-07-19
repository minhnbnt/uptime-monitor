## ADDED Requirements

### Requirement: Count servers by status over gRPC
server-service SHALL expose a gRPC RPC `CountServersByStatus` on `server.v1.ServerService` that accepts a `user_id` and returns the total number of servers, the number online, and the number offline for that user.

#### Scenario: Successful count for a user with servers
- **WHEN** a gRPC caller invokes `CountServersByStatus` with a valid `user_id`
- **THEN** the server returns `total`, `online`, and `offline` counts for that user's servers

#### Scenario: User with no servers
- **WHEN** a gRPC caller invokes `CountServersByStatus` for a `user_id` that owns no servers
- **THEN** the server returns `total = 0`, `online = 0`, `offline = 0` without error

### Requirement: gRPC count mirrors HTTP count behavior
The gRPC `CountServersByStatus` RPC SHALL return the same counts as the existing HTTP `GET /api/v1/servers/count` endpoint for the same user.

#### Scenario: Parity between gRPC and HTTP
- **WHEN** the same `user_id` is queried via gRPC `CountServersByStatus` and via HTTP `/api/v1/servers/count`
- **THEN** both responses contain identical `total`, `online`, and `offline` values

### Requirement: HTTP endpoint remains available
server-service SHALL continue to serve `GET /api/v1/servers/count` over HTTP with the existing `ServerCountResponse` shape.

#### Scenario: External HTTP client still works
- **WHEN** an external client calls `GET /api/v1/servers/count` with a valid `X-User-ID` header
- **THEN** the server returns the same `{total, online, offline}` JSON response as before
