## ADDED Requirements

### Requirement: Server object exposes monitor status
The system SHALL include a `monitor_status` field (string enum: `ON` or `OFF`) on the `ServerObject` response for all server read endpoints (list, get, search).

#### Scenario: List servers returns monitor status
- **WHEN** a client calls `GET /api/v1/servers`
- **THEN** each item in `data` contains a `monitor_status` field set to `ON` or `OFF` reflecting the server's current status from `ontime-service`

#### Scenario: Get single server returns monitor status
- **WHEN** a client calls `GET /api/v1/servers/{id}`
- **THEN** the `data` object contains a `monitor_status` field reflecting the server's current status

#### Scenario: Search servers returns monitor status
- **WHEN** a client calls `GET /api/v1/servers/search`
- **THEN** each result in `data` contains a `monitor_status` field

### Requirement: Monitor status sourced from status service
The system SHALL derive `monitor_status` from `ontime-service`'s `StatusService.GetCurrentStatuses`, keyed by the server's endpoint ID. `server-service` SHALL NOT store server events.

#### Scenario: Status mapped by endpoint ID
- **WHEN** the service enriches a server that has an associated endpoint
- **THEN** the status is looked up using that endpoint's ID in `GetCurrentStatuses`

#### Scenario: Server without endpoint
- **WHEN** a server has no associated endpoint
- **THEN** `monitor_status` is `null` (status unknown)

### Requirement: Monitor status is best-effort
If the `GetCurrentStatuses` call fails, the system SHALL still return the server response with `monitor_status` as `null` and log a warning, rather than failing the request.

#### Scenario: Status service unavailable
- **WHEN** `ontime-service`'s `StatusService` is unreachable or returns an error during enrichment
- **THEN** the server response is returned successfully with `monitor_status` set to `null`

### Requirement: Newly created server has null status
The system SHALL return `monitor_status` as `null` for a server that has no recorded events yet.

#### Scenario: Fresh server
- **WHEN** a server was just created and has no uptime events recorded
- **THEN** `monitor_status` is `null`
