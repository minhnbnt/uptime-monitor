## ADDED Requirements

### Requirement: Notification-service consumes server-service over gRPC
notification-service SHALL call server-service exclusively over gRPC for both listing servers and counting servers by status. It MUST NOT use HTTP for these calls.

#### Scenario: Counting server status via gRPC
- **WHEN** the digest workflow needs server status counts for a user
- **THEN** notification-service invokes the gRPC `CountServersByStatus` RPC and uses the returned `total`, `online`, `offline`

#### Scenario: Listing servers via gRPC
- **WHEN** the digest workflow needs the list of a user's servers
- **THEN** notification-service invokes the gRPC `ListServers` RPC and maps the result into its domain `Server` type

### Requirement: Removed HTTP server client
notification-service SHALL NOT retain an HTTP client for server-service. The `serverclient` HTTP implementation (base URL, `net/http`, JSON decode) MUST be removed.

#### Scenario: No HTTP server client remains
- **WHEN** the codebase is built after the migration
- **THEN** there is no HTTP-based server-service client in notification-service

### Requirement: gRPC client configuration
notification-service SHALL read its server-service gRPC address from a `grpc.server_addr` config value (plain `host:port`, default `localhost:50051`).

#### Scenario: Default gRPC address
- **WHEN** no `grpc.server_addr` is provided
- **THEN** notification-service defaults to `localhost:50051`, matching server-service's gRPC port

### Requirement: Observability of gRPC calls
notification-service SHALL log each gRPC call to server-service (request sent, success, and failures with error details) at debug/error level.

#### Scenario: Failure is logged
- **WHEN** a gRPC call to server-service fails
- **THEN** notification-service logs the error with the target URL/user and returns a wrapped error
