## Why

The scheduler's `ClaimDueTasks` fans out to every Redis shard *internally* (one
goroutine, `shardCount` parallel `claimScript` calls, then a merge + trim) even
though sharding is only used to spread load. This adds needless cross-shard
merge logic, lock-score math, and a wait-group for no benefit: each shard is an
independent queue and can be owned by its own goroutine. Simplifying to
per-shard goroutines removes the merge complexity and makes each loop
wake exactly when its own shard is due.

## What Changes

- `ZSetScheduleRepository.ClaimDueTasks(ctx, limit)` is replaced by
  `ClaimDueTasksForShard(ctx, shardID uint, limit int64)` — a single
  `claimScript` call on `scheduler:queue:<shardID>`, with no internal fan-out,
  no `waitGroup`, and no cross-shard merge/trim.
- `ZSetWorkerRunner.RunZSetWorker` spawns one goroutine per shard
  (`cfg.Redis.SchedulerShards`), each calling `LoopService.Run` with its own
  `shardID` and per-shard `claimLimit`.
- `LoopService.Run` takes `shardID uint` and `claimLimit int64` and loops on
  that single shard only.
- Config gains `scheduler_claim_limit` (default `10`); the per-shard claim limit
  is read from config instead of the hard-coded `defaultClaimLimit = 50`.
- The `claimLock` mechanism and write-side shard routing
  (`schedulerShardKey`, `RegisterBatch`, `Unregister`, `UpdateBatch`) are
  unchanged.

## Capabilities

### New Capabilities

<!-- No new capabilities; the change is an internal restructuring of the scheduler loop. -->

### Modified Capabilities

<!-- No spec-level (requirement) behavior changes. Claim semantics, lock
     protection, and write routing are preserved. -->

## Impact

- `ping-service/internal/infrastructure/scheduler/zset.go` — claim API change.
- `ping-service/internal/service/zsetloop.go` — `Run` signature + loop body.
- `ping-service/internal/handler/zsetworker.go` — goroutine spawning + config inject.
- `ping-service/internal/app/http.go` — `RunZSetWorker` passthrough (unchanged signature).
- `ping-service/internal/config/redis.go` + `viper.go` — new `SchedulerClaimLimit` config + env.
- Tests: remove obsolete cross-shard merge tests
  (`TestClaimDueTasksMergesAcrossShards`,
  `TestClaimDueTasksEarliestNextAcrossShards`,
  `TestClaimDueTasksCappedAtLimit`); adapt `ClaimDueTasksZeroLimit` and lock tests
  to the per-shard signature; add a per-shard isolation test.
