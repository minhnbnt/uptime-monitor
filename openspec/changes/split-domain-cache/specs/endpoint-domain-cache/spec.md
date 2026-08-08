## Purpose

Caches the resolved Pod IP for HTTP-DNS checks in a cache keyed by k8s object identity `(namespace, kind, object_id)`, so servers pointing at the same Pod share one k8s API resolution and a stale domain can be invalidated without touching the server meta cache. Service and StatefulSet domains are computed DNS strings and are never cached.

## ADDED Requirements

### Requirement: Pod domain cache keyed by object identity
ping-service SHALL store the resolved domain of a Pod-kind endpoint in a cache key derived from the k8s object identity `(namespace, kind, object_id)`, so all servers pointing at the same Pod share a single cache entry. Only Pod-kind endpoints SHALL be cached.

#### Scenario: Multiple servers share one entry
- **WHEN** two servers target the same `(namespace, kind, object_id)` where kind is Pod
- **THEN** resolving the domain for one server populates the entry for both, and the second resolve does not call the k8s API again within the cache TTL

### Requirement: Lazy domain resolution at URL resolve time
When an HTTP URL is resolved for a ping, ping-service SHALL consult the domain cache first; on a miss it SHALL resolve the domain via the k8s client and write it back to the cache (best-effort, skipping empty values).

#### Scenario: Cache hit
- **WHEN** a cached domain exists for the object
- **THEN** the URL is built from the cached domain without calling the k8s API

#### Scenario: Cache miss
- **WHEN** no cached domain exists for the object
- **THEN** ping-service resolves the domain via the k8s client, stores it in the cache, and builds the URL from it

#### Scenario: Fresh resolution for stale detection
- **WHEN** stale-domain detection needs a current domain to compare against the cached one
- **THEN** ping-service resolves directly via the k8s client, bypassing the cache

### Requirement: Service and StatefulSet domains resolved without cache
ping-service SHALL resolve Service and StatefulSet domains directly as computed DNS strings at URL resolve time, without consulting or populating the domain cache.

#### Scenario: Service URL resolve
- **WHEN** an HTTP URL is resolved for a Service-kind endpoint
- **THEN** the domain is computed from the object name and namespace without a cache lookup and without a k8s API call

#### Scenario: StatefulSet URL resolve
- **WHEN** an HTTP URL is resolved for a StatefulSet-kind endpoint
- **THEN** the domain is computed from the object name and namespace without a cache lookup and without a k8s API call

### Requirement: Meta cache excludes resolved k8s fields
The server meta cache SHALL store only server identity, scheduling parameters (interval, timeout), and HTTP config; it SHALL NOT store the resolved domain or the label selector.

#### Scenario: Meta cache write
- **WHEN** a server batch is written into the meta cache
- **THEN** the entry contains identity, interval, timeout, and HTTP config fields only

#### Scenario: Meta cache read
- **WHEN** a server is read from the meta cache
- **THEN** it contains no cached domain or label selector value

### Requirement: Label selector resolved live
ping-service SHALL NOT cache the workload label selector; a check that needs a selector SHALL resolve it from the k8s API at check time.

#### Scenario: Workload check without cached selector
- **WHEN** a workload server is checked and no selector is cached
- **THEN** the check resolves the selector from the k8s API before listing pods

### Requirement: Stale domain invalidates only the domain cache
When ping-service detects that the resolved domain differs from the cached one, it SHALL delete only the domain cache entry for that object, leaving the meta cache (identity, scheduling, HTTP config) intact.

#### Scenario: Stale Pod IP detected
- **WHEN** an HTTP ping to a cached Pod IP fails and a fresh resolve returns a different IP
- **THEN** the domain cache entry for the object is deleted, the meta cache is untouched, and the event is skipped so the next check re-resolves
