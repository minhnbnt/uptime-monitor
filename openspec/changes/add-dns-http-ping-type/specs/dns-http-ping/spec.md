## Purpose

Enables HTTP health checks against K8s resources (Service, Pod, StatefulSet) by resolving their in-cluster DNS names and making HTTP GET requests to user-defined ports and paths, alongside the existing pod-status monitoring.

## ADDED Requirements

### Requirement: PingType proto enum
The system SHALL define a proto enum `PingType` with values `PING_TYPE_POD_STATUS` (0) and `PING_TYPE_HTTP_DNS` (1).
When `PingType` is `PING_TYPE_POD_STATUS`, the existing K8s pod container readiness check SHALL be used.
When `PingType` is `PING_TYPE_HTTP_DNS`, an HTTP GET health check via K8s DNS resolution SHALL be used.

#### Scenario: POD_STATUS enum value
- **WHEN** a Server has no `server_http_configs` record
- **THEN** the ping-service SHALL use `PingType = PING_TYPE_POD_STATUS`

#### Scenario: HTTP_DNS enum value
- **WHEN** a Server has a `server_http_configs` record with `port = 8080`, `endpoint_path = "/health"`
- **THEN** the ping-service SHALL use `PingType = PING_TYPE_HTTP_DNS` with `HttpDnsConfig{port: 8080, endpoint_path: "/health"}`

### Requirement: HttpDnsConfig proto message
The system SHALL define a proto message `HttpDnsConfig` containing `port` (int32) and `endpoint_path` (string).
This message SHALL be an optional field on both `EndpointData` and `PingRequest`.

#### Scenario: HttpDnsConfig populated
- **WHEN** a Server has http-dns configuration
- **THEN** the `EndpointData` SHALL include a populated `HttpDnsConfig`

### Requirement: server_http_configs table
The system SHALL store HTTP-DNS configuration in a separate table `server_http_configs`:
- `server_id` (uint, PK, FK → servers.id ON DELETE CASCADE)
- `port` (int, NOT NULL)
- `endpoint_path` (varchar, DEFAULT '')
- `created_at`, `updated_at`

The existence of a record SHALL determine the PingType: record exists → `HTTP_DNS`, no record → `POD_STATUS`.

#### Scenario: Create server with http_config
- **WHEN** a new Server is created with `http_config = {port: 8080, endpoint_path: "/health"}`
- **THEN** the system SHALL insert a `server_http_configs` record

#### Scenario: Delete http_config via update
- **WHEN** a Server is updated to remove the `http_config` object
- **THEN** the system SHALL delete the corresponding `server_http_configs` record

### Requirement: DNS resolution for http-dns pings
The system SHALL resolve in-cluster DNS names for K8s resource types when `PingType = HTTP_DNS`:
- Kind `Service`: SHALL use `<object_id>.<namespace>.svc.cluster.local`
- Kind `Pod`: SHALL retrieve the Pod's IP from the K8s API and use it directly
- Kind `StatefulSet`: SHALL use `<object_id>-0.<object_id>.<namespace>.svc.cluster.local`

#### Scenario: Service DNS resolution
- **WHEN** a Server has `kind = "Service"`, `object_id = "my-api"`, `namespace = "default"`, `PingType = HTTP_DNS`
- **THEN** the system SHALL resolve the DNS name `my-api.default.svc.cluster.local`

#### Scenario: Pod IP resolution
- **WHEN** a Server has `kind = "Pod"`, `object_id = "my-pod-xyz"`, `namespace = "default"`, `PingType = HTTP_DNS`
- **THEN** the system SHALL retrieve the Pod's IP from the K8s API

#### Scenario: StatefulSet DNS resolution
- **WHEN** a Server has `kind = "StatefulSet"`, `object_id = "my-sts"`, `namespace = "default"`, `PingType = HTTP_DNS`
- **THEN** the system SHALL resolve the DNS name `my-sts-0.my-sts.default.svc.cluster.local`

### Requirement: HTTP health check execution
When `PingType = HTTP_DNS`, the system SHALL make an HTTP GET request to `http://<resolved-dns>:<port>/<endpoint_path>`.
The HTTP request SHALL respect the configured timeout.
A response with status code 2xx SHALL be considered healthy (ON).
Any other response or connection error SHALL be considered unhealthy (OFF).

#### Scenario: Successful health check
- **WHEN** the HTTP GET returns status 200
- **THEN** the endpoint status SHALL be reported as ON

#### Scenario: Failed health check
- **WHEN** the HTTP GET returns status 500
- **THEN** the endpoint status SHALL be reported as OFF

#### Scenario: Timeout during health check
- **WHEN** the HTTP GET exceeds the configured timeout
- **THEN** the endpoint status SHALL be reported as OFF

### Requirement: Service kind in API enum
The `kind` field SHALL accept `"Service"` as a valid value in addition to the existing kinds (Pod, Deployment, StatefulSet, DaemonSet, ReplicaSet).

#### Scenario: Create Server with kind Service
- **WHEN** a Server is created with `kind = "Service"`
- **THEN** the system SHALL accept and store the record

### Requirement: New fields propagated through all layers
The `PingType`, `HttpDnsConfig` SHALL be propagated through:
- gRPC `EndpointData` messages (server-service → ping-service)
- gRPC `PingRequest` messages (on-demand ping)
- Redis endpoint cache
- CDC (Debezium) change events

#### Scenario: EndpointData includes PingType and HttpDnsConfig
- **WHEN** ping-service calls GetEndpoints
- **THEN** the returned EndpointData SHALL include `PingType` enum and `HttpDnsConfig` when applicable
