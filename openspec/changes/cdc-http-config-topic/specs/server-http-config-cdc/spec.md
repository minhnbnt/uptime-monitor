## Purpose

Keeps ping-service's scheduler meta cache in sync with HTTP config changes (create, update, delete) on the server_http_configs table via the CDC stream uptime.public.server_http_configs, so ping behavior reflects the current HTTP config without waiting for cache expiry.

## ADDED Requirements

### Requirement: Consume server_http_configs CDC stream
The ping-service SHALL consume the `uptime.public.server_http_configs` Redis stream alongside the existing `uptime.public.servers` stream, in the same consumer process and group lifecycle.

#### Scenario: Stream is consumed
- **WHEN** ping-service runs and the stream has pending messages
- **THEN** messages from both `uptime.public.servers` and `uptime.public.server_http_configs` are processed and acknowledged

### Requirement: Apply HTTP config create/update events
When an insert or update event (op `c`, `r`, or `u`) arrives for a server_http_configs row, ping-service SHALL write the server's HTTP config fields (port, endpoint path, expected code, body check expression, method) and the HTTP-config marker into the scheduler meta cache for that server ID.

#### Scenario: HTTP config created
- **WHEN** an insert event for server_http_configs arrives
- **THEN** the cache entry for that server ID has an HTTP-config marker and the config fields from the event

#### Scenario: HTTP config updated
- **WHEN** an update event for server_http_configs arrives
- **THEN** the cache entry for that server ID reflects the new config fields

#### Scenario: Event without payload
- **WHEN** an insert/update event has no `after` payload
- **THEN** the event is acknowledged without modifying the cache

### Requirement: Apply HTTP config delete events
When a delete event (op `d`) arrives for a server_http_configs row, ping-service SHALL remove the HTTP-config marker and HTTP config fields from the cache entry for that server ID, causing the server to be checked as pod-status.

#### Scenario: HTTP config deleted
- **WHEN** a delete event for server_http_configs arrives
- **THEN** the cache entry for that server ID no longer has the HTTP-config marker or config fields

#### Scenario: Delete without before payload
- **WHEN** a delete event has neither `before` nor `after` carrying the server ID
- **THEN** the event is not acknowledged and an error is logged

### Requirement: HTTP config flows into scheduled pings
A server whose cache entry carries an HTTP-config marker SHALL be pinged as HTTP_DNS (resolved URL + response check); a server without the marker SHALL be checked via pod/workload status.

#### Scenario: Marker present after CDC update
- **WHEN** a server's HTTP config is upserted via CDC
- **THEN** subsequent scheduled checks of that server use the HTTP_DNS path with the cached config

#### Scenario: Marker cleared after CDC delete
- **WHEN** a server's HTTP config is deleted via CDC
- **THEN** subsequent scheduled checks of that server use the pod-status path
