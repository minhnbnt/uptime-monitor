## 1. K8s Client Infrastructure (ping-service)

- [x] 1.1 Add `k8s.io/client-go` dependency to ping-service go.mod
- [x] 1.2 Create `internal/infrastructure/k8sclient/client.go` — K8sClient interface with `CheckPodStatus(ctx, namespace, kind, objectID, containerName) (bool, error)`, in-cluster config setup
- [x] 1.3 Implement pod status check: GET pod by name for Pod kind, LIST pods with label selector for Deployment/StatefulSet/DaemonSet/ReplicaSet
- [x] 1.4 Implement running logic: pod.phase == Running AND all containers (or specified container) ready == true
- [x] 1.5 Register K8sClient in DI injector

## 2. Domain Model (ping-service)

- [x] 2.1 Rewrite `internal/domain/endpoint.go` — rename to server.go, replace HTTP fields (URL, Method, ExpectedCode, BodyCheckExpr) with k8s fields (Namespace, Kind, ObjectID, ContainerName), keep Interval/Timeout
- [x] 2.2 Update `internal/domain/serverevent.go` — rename EndpointID → ServerID
- [x] 2.3 Update `internal/service/task.go` — PingTask.Endpoint → PingTask.Server

## 3. Infrastructure — Remove HTTP, Add K8s

- [x] 3.1 Delete `internal/infrastructure/ping.go` (HTTP PingClient) — kept as dead code per user request
- [x] 3.2 Delete `internal/infrastructure/bodychecker.go` (expr-lang body evaluator) — kept as dead code per user request
- [x] 3.3 Delete `internal/service/responsechecker.go` (HTTP response validation) — kept as dead code per user request
- [x] 3.4 Update `internal/infrastructure/recordstatus.go` — change event recording to use ServerID

## 4. Service Layer (ping-service)

- [x] 4.1 Rewrite `internal/service/pingloop.go` — replace HTTP ping + response check with K8sClient.CheckPodStatus, update log fields (remove url/method/expected_code, add namespace/kind/object_id)
- [x] 4.2 Update `internal/service/endpointevent.go` — rename handler interface to use domain.Server instead of domain.Endpoint

## 5. Scheduler & Cache (ping-service)

- [x] 5.1 Update `internal/infrastructure/scheduler/endpointcache.go` — cache k8s fields (namespace, kind, object_id, container_name, interval_ns, timeout_ns) instead of HTTP fields, rename mapToEndpoint → mapToServer
- [x] 5.2 Update `internal/infrastructure/scheduler/endpointprovider.go` — rename methods and types to use Server instead of Endpoint
- [x] 5.3 Update `internal/infrastructure/scheduler/zset.go` — update Register/Unregister to use domain.Server
- [x] 5.4 Update `internal/infrastructure/scheduler/zsetClaimer.go` — no field changes needed, verify compatibility
- [x] 5.5 Update `internal/infrastructure/scheduler/scoreupdater.go` — no field changes needed, verify compatibility

## 6. gRPC Server (ping-service)

- [x] 6.1 Rewrite `internal/infrastructure/grpcserver/ping_server.go` — accept k8s fields (namespace, kind, object_id, container_name, timeout_ms), use K8sClient instead of HTTP PingClient, return running (bool)
- [x] 6.2 Delete ResponseChecker dependency from PingServer

## 7. gRPC Client (ping-service)

- [x] 7.1 Update `internal/infrastructure/grpcclient/endpoint_client.go` — map new EndpointData fields (namespace, kind, object_id, container_name) to domain.Server

## 8. Debezium CDC Consumer (ping-service)

- [x] 8.1 Update `internal/infrastructure/redis/redisconsumer.go` — change streamKey from `uptime.public.endpoints` to `uptime.public.servers`
- [x] 8.2 Update `internal/infrastructure/redis/processor.go` — rename debeziumEndpointData → debeziumServerData with k8s fields, update toDomain mapping

## 9. Proto — Event Service (common module)

- [x] 9.1 Update `common/proto/event/v1/event_service.proto` — rename endpoint_id → server_id in RecordEventRequest, GetCurrentStatusesRequest, EndpointStatus → ServerStatus
- [x] 9.2 Regenerate proto code (`buf generate` in common/proto)

## 10. OnTime Service — Domain & Repository

- [x] 10.1 Update `ontime-service/internal/domain/serverevent.go` — rename EndpointID → ServerID
- [x] 10.2 Update `ontime-service/internal/infrastructure/repository/serverevent.go` — change endpoint_id → server_id in queries
- [ ] 10.3 DB migration: rename server_events.endpoint_id → server_id (ALTER TABLE)

## 11. OnTime Service — Handlers

- [x] 11.1 Update event handler (RecordEvent) — use server_id from proto request
- [x] 11.2 Update status handler (GetCurrentStatuses) — query by server_ids
- [x] 11.3 Update ontime handler — join on server_id instead of endpoint_id

## 12. K8s RBAC Manifests

- [x] 12.1 Create Helm RBAC template — ServiceAccount, ClusterRole (pods/get, pods/list, deployments/get, statefulsets/get, daemonsets/get, replicasets/get), ClusterRoleBinding
- [x] 12.2 Update ping-service Deployment spec to use the ServiceAccount

## 13. DI & App Setup

- [x] 13.1 Update ping-service `internal/app/app.go` — remove HTTP PingClient registration, add K8sClient registration
- [x] 13.2 Update ontime-service DI if needed for new proto types

## 14. Verification

- [x] 14.1 Run `go build ./...` on ping-service
- [x] 14.2 Run `go build ./...` on ontime-service
- [x] 14.3 Run `golangci-lint run` on both services
- [x] 14.4 Run existing tests on both services
- [ ] 14.5 Verify ping-service starts, connects to k8s API, and checks pod status
