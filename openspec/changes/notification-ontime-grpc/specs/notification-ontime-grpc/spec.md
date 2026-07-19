## ADDED Requirements

### Requirement: Notification-service consumes ontime over gRPC
notification-service SHALL fetch server uptime statistics from ontime-service exclusively over gRPC. It MUST NOT use HTTP for these calls.

#### Scenario: Fetching ontime stats via gRPC
- **WHEN** the digest workflow needs per-server daily uptime for a user
- **THEN** notification-service invokes the gRPC `GetServersOntime` RPC and maps the result into its domain ontime-stats type

### Requirement: Removed HTTP ontime client
notification-service SHALL NOT retain an HTTP client for ontime-service. The `ontimeclient` HTTP implementation (base URL, `net/http`, JSON encode/decode, `/batch` route) MUST be removed.

#### Scenario: No HTTP ontime client remains
- **WHEN** the codebase is built after the migration
- **THEN** there is no HTTP-based ontime-service client in notification-service

### Requirement: gRPC client configuration
notification-service SHALL read its ontime-service gRPC address from a `grpc.event_addr` (or equivalent) config value (plain `host:port`, default `ontime-service:50052`).

#### Scenario: Default gRPC address
- **WHEN** no ontime gRPC address is provided
- **THEN** notification-service defaults to `ontime-service:50052`, matching ontime-service's gRPC port

### Requirement: Observability of gRPC calls
notification-service SHALL log each gRPC call to ontime-service (request sent, success, and failures with error details) at debug/error level.

#### Scenario: Failure is logged
- **WHEN** a gRPC call to ontime-service fails
- **THEN** notification-service logs the error with the target and user and returns a wrapped error
