## Context

`ping-service` schedules endpoint pings using Redis sorted sets sharded by
`endpointID` hash (`scheduler:queue:<shardID>`). Today a single `LoopService.Run`
goroutine drives all shards: `ClaimDueTasks` spins up `shardCount` parallel
`claimScript` calls, merges their `due` lists, computes the earliest `next`
across shards, trims to `limit`, then `runIteration` + `sleep`.

Each shard is already an independent queue. The only reason multiple claimers
must coordinate is *cross-process* (multiple ping-service replicas) — handled by
the `claimLock` score bump, not by the in-process fan-out. The in-process
merge adds complexity with no benefit.

## Goals / Non-Goals

**Goals:**
- One goroutine per shard; each owns `scheduler:queue:<shardID>` exclusively.
- Drop the internal `shardCount` fan-out, `waitGroup`, and cross-shard merge/trim.
- Per-shard claim limit driven by config (default `10`).
- Preserve `claimLock` cross-replica double-claim protection.
- Preserve write-side shard routing (`schedulerShardKey`) and `GenerateOffset`.

**Non-Goals:**
- Changing the Redis key schema or shard routing hash.
- Changing `claimScript` Lua logic (it already operates on a single key).
- Dynamic shard rebalancing / re-sharding at runtime.
- Horizontal scaling beyond `SchedulerShards` goroutines per process.

## Decisions

**D1. Who spawns the goroutines → `ZSetWorkerRunner` (option b).**
The worker runner already owns the process-level "run" boundary and the
`dueHandler` closure. It reads `cfg.Redis.SchedulerShards`, loops `shardID`, and
launches `go loopService.Run(ctx, shardID, claimLimit, handler)` per shard.
Rationale: keeps the scheduler repo unaware of how many goroutines exist; the
repo only needs the `shardID` it is told to claim. Alternative (spawning inside
`LoopService.Run`) was rejected to keep the loop single-shard-agnostic and the
"N workers" concept at the process boundary.

**D2. `ClaimDueTasks` → `ClaimDueTasksForShard(ctx, shardID, limit)`.**
Signature gains an explicit `shardID uint`; the body is a single
`claimScript.Run(ctx, client, [shardKey(shardID)], now, limit, lockMs)`.
Removed: `perShardLimit = limit/shardCount + 1`, `shardResult` slice,
`waitGroup`, merge loop, and `len(allDue) > limit` trim (a single shard already
returns at most `limit`). A new helper `shardKey(shardID)` builds the key.

**D3. `shardCount` stays in the repo — but only for writes.**
`RegisterBatch`, `Unregister`, and `ScoreUpdater.UpdateBatch` still use
`schedulerShardKey(shardCount, endpointID)` to route writes. Reads now take an
explicit `shardID`. So `shardCount` remains a constructor param; it is no longer
used on the claim path.

**D4. Per-shard claim limit is config (`SchedulerClaimLimit`, default 10).**
Replaces the hard-coded `defaultClaimLimit = 50`. Throughput per iteration
becomes `claimLimit × shardCount` (was ~`limit` total). This is intentional and
tunable; the user set the desired value to `10`.

**D5. `LoopService.Run(ctx, shardID, claimLimit, handler)`.**
Loops while `ctx.Err() == nil`, calling `ClaimDueTasksForShard` with its own
`shardID`/`claimLimit`, then `runIteration` and `sleepCtx(getSleepDuration(next,
hasNext))`. `runIteration` and `getSleepDuration` are unchanged — they already
operate on a single `next`.

## Risks / Trade-offs

- [Higher total claim throughput] → `claimLimit(10) × shardCount` tasks per
  iteration vs the old ~`limit` total. Mitigation: tunable via
  `SchedulerClaimLimit`; start at `10` as requested.
- [Uneven shard load] → one hot shard's goroutine may lag while another idles.
  Mitigation: acceptable; hashing spreads endpoints, and each shard locks
  independently so lag is bounded per shard, not global.
- [Graceful shutdown] → N goroutines must all stop. Mitigation: they share the
  `ctx`; `Run` loops on `ctx.Err() == nil`, so cancelling `ctx` stops all.
- [Test surface for merge removed] → cross-shard merge tests become invalid.
  Mitigation: delete them; add a per-shard isolation test asserting a goroutine
  claims only its own `scheduler:queue:<id>`.

## Migration Plan

1. Add `SchedulerClaimLimit` to `RedisConfig` + defaults (`10`) + env binding.
2. Update repo claim API + add `shardKey`.
3. Update `LoopService.Run` signature + `LoopRunner` interface.
4. Update `ZSetWorkerRunner` to inject `config` and spawn per-shard goroutines.
5. Update/remove affected tests; `go test ./...` and `go build ./...`.
6. Rollout is config-only; no Redis schema change. Rollback = revert binary;
   `claimLock` semantics unchanged so old/new claimers interoperate on the same
   keys during a rolling deploy.

## Open Questions

<!-- None outstanding; all design decisions resolved during exploration. -->
