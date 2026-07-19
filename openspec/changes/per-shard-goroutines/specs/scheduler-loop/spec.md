## ADDED Requirements

### Requirement: Per-shard scheduler goroutine ownership
The scheduler SHALL run one independent goroutine per configured shard
(`redis.scheduler_shards`), where each goroutine claims due tasks exclusively
from its own `scheduler:queue:<shardID>` Redis sorted set. The scheduler SHALL
NOT fan out internally across shards within a single claim call.

#### Scenario: Each shard claimed by its own goroutine
- **WHEN** the worker runner starts with `scheduler_shards = N`
- **THEN** exactly N goroutines are spawned, goroutine `i` claims only from `scheduler:queue:i`

#### Scenario: No internal cross-shard merge
- **WHEN** a goroutine claims due tasks
- **THEN** it performs a single claim on its own shard key and returns results for that shard only, without merging other shards

### Requirement: Config-driven per-shard claim limit
The number of due tasks claimed per shard per iteration SHALL be controlled by
the `redis.scheduler_claim_limit` configuration value, which MUST default to `10`
when unset.

#### Scenario: Default claim limit applied
- **WHEN** `scheduler_claim_limit` is not provided
- **THEN** each shard goroutine claims at most 10 due tasks per iteration

#### Scenario: Explicit claim limit applied
- **WHEN** `scheduler_claim_limit` is set to a value `L`
- **THEN** each shard goroutine claims at most `L` due tasks per iteration

### Requirement: Cross-replica claim lock preserved
The scheduler SHALL retain the existing `claimLock` score-bump mechanism so that
concurrent claimers across different process replicas cannot double-claim the
same task on a shard.

#### Scenario: Lock prevents re-claim within window
- **WHEN** a task is claimed by one claimer
- **THEN** another claimer cannot re-claim the same task until the lock score expires
