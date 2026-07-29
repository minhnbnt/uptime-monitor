## ADDED Requirements

### Requirement: K8s pod status checking
The system SHALL query pod status via the k8s API using client-go. A server is "running" if the target pod exists, has phase == Running, and all containers (or the specified container) have ready == true.

#### Scenario: Pod is running with all containers ready
- **WHEN** the system checks a Pod with namespace="default", kind="Pod", object_id="web-app"
- **AND** the pod exists with phase=Running and all containers ready=true
- **THEN** the check returns running=true

#### Scenario: Pod exists but not ready
- **WHEN** the system checks a Pod with namespace="default", kind="Pod", object_id="web-app"
- **AND** the pod exists with phase=Running but a container has ready=false
- **THEN** the check returns running=false

#### Scenario: Pod not found
- **WHEN** the system checks a Pod with namespace="default", kind="Pod", object_id="nonexistent"
- **AND** no pod with that name exists in the namespace
- **THEN** the check returns running=false

#### Scenario: Pod in failed state
- **WHEN** the system checks a Pod with namespace="default", kind="Pod", object_id="crashed-app"
- **AND** the pod exists with phase=Failed
- **THEN** the check returns running=false

### Requirement: Deployment/StatefulSet/DaemonSet/ReplicaSet status checking
The system SHALL resolve workload kinds (Deployment, StatefulSet, DaemonSet, ReplicaSet) to their constituent pods using label selectors, then check pod status.

#### Scenario: Deployment with running pods
- **WHEN** the system checks a Deployment with namespace="prod", kind="Deployment", object_id="web-app"
- **AND** the deployment has pods with the matching app label
- **AND** at least one pod is running with all containers ready
- **THEN** the check returns running=true

#### Scenario: Deployment with no running pods
- **WHEN** the system checks a Deployment with namespace="prod", kind="Deployment", object_id="web-app"
- **AND** all matching pods are not ready or pending
- **THEN** the check returns running=false

#### Scenario: Deployment not found
- **WHEN** the system checks a Deployment with namespace="prod", kind="Deployment", object_id="nonexistent"
- **AND** no deployment with that name exists
- **THEN** the check returns running=false with an error message

### Requirement: Container-specific status check
When container_name is specified, the system SHALL only check that specific container's readiness status.

#### Scenario: Specific container is ready
- **WHEN** the system checks a Pod with container_name="nginx"
- **AND** the nginx container has ready=true
- **THEN** the check returns running=true regardless of other containers

#### Scenario: Specific container not ready
- **WHEN** the system checks a Pod with container_name="nginx"
- **AND** the nginx container has ready=false
- **THEN** the check returns running=false

#### Scenario: Specific container not found in pod
- **WHEN** the system checks a Pod with container_name="nonexistent"
- **AND** no container with that name exists in the pod
- **THEN** the check returns running=false with an error message

### Requirement: Ping gRPC server accepts k8s fields
The Ping gRPC service SHALL accept namespace, kind, object_id, container_name, timeout_ms in PingRequest and return running (bool) in PingResponse.

#### Scenario: On-demand ping via gRPC
- **WHEN** server-service calls Ping via gRPC with namespace="default", kind="Pod", object_id="test-pod"
- **THEN** the response contains running=true or running=false based on pod status

#### Scenario: On-demand ping with timeout
- **WHEN** server-service calls Ping via gRPC with timeout_ms=5000
- **THEN** the k8s API call is cancelled after 5 seconds

### Requirement: ZSet scheduler stores k8s identity
The ZSet scheduler cache SHALL store namespace, kind, object_id, container_name instead of URL, method, expected_code.

#### Scenario: Cache stores k8s fields
- **WHEN** an endpoint is cached in Redis
- **THEN** the cache hash contains namespace, kind, object_id, container_name, interval_ns fields

#### Scenario: Cache miss falls back to gRPC
- **WHEN** the scheduler cannot find an endpoint in Redis cache
- **THEN** it fetches the endpoint data from server-service via gRPC GetEndpoints

### Requirement: Debezium CDC consumes from servers topic
The ping-service Redis stream consumer SHALL listen to `uptime.public.servers` instead of `uptime.public.endpoints`.

#### Scenario: Server created event
- **WHEN** a new server is created in server-service
- **AND** Debezium publishes a CDC event to `uptime.public.servers`
- **THEN** ping-service registers the server in the ZSet scheduler

#### Scenario: Server updated event
- **WHEN** a server is updated in server-service
- **AND** Debezium publishes a CDC event to `uptime.public.servers`
- **THEN** ping-service invalidates the cache and re-registers in the scheduler

#### Scenario: Server deleted event
- **WHEN** a server is deleted from server-service
- **AND** Debezium publishes a CDC event to `uptime.public.servers`
- **THEN** ping-service unregisters the server from the ZSet scheduler

### Requirement: Status recording by server_id
The ping-service SHALL record status events using server_id instead of endpoint_id. The ontime-service SHALL store events with server_id.

#### Scenario: Record status change
- **WHEN** a server's status changes from ON to OFF
- **THEN** ping-service calls RecordEvent with server_id (not endpoint_id)

#### Scenario: Query current statuses
- **WHEN** server-service calls GetCurrentStatuses with server_ids
- **THEN** ontime-service returns the latest status for each server_id

#### Scenario: Dedup consecutive same-status events
- **WHEN** a server is checked and status is still ON (same as last recorded)
- **THEN** no new event is written to server_events

### Requirement: Server domain model in ping-service
The ping-service Domain model SHALL be renamed from Endpoint to Server with k8s identity fields: ID, ServerID, Namespace, Kind, ObjectID, ContainerName, Interval, Timeout.

#### Scenario: Server has k8s fields
- **WHEN** a server is loaded from the cache or gRPC
- **THEN** the domain.Server struct contains namespace, kind, object_id, container_name fields

#### Scenario: No HTTP fields
- **WHEN** a server is loaded
- **THEN** the domain.Server struct does NOT contain url, method, expected_code, or body_check_expr fields
