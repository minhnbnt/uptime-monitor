## 1. Config

- [x] 1.1 Add `SchedulerClaimLimit int` with `mapstructure:"scheduler_claim_limit"` to `RedisConfig` in `internal/config/redis.go`.
- [x] 1.2 Add default `"redis.scheduler_claim_limit": 10` in `setDefaults` and env binding `"redis.scheduler_claim_limit": "REDIS_SCHEDULER_CLAIM_LIMIT"` in `bindEnvVars` (`internal/config/viper.go`).

## 2. Scheduler Repository

- [x] 2.1 Add `shardKey(shardID uint) string` helper returning `fmt.Sprintf("%s:%d", schedulerQueuePrefix, shardID)`.
- [x] 2.2 Replace `ClaimDueTasks(ctx, limit)` with `ClaimDueTasksForShard(ctx, shardID uint, limit int64)` — single `claimScript.Run` on `shardKey(shardID)`, dropping `perShardLimit`, `shardResult`, `waitGroup`, cross-shard merge, and trim. Keep return shape `(due, next, hasNext, err)`.
- [x] 2.3 Remove the now-unused `claimLock`-related `perShardLimit` math; keep `lockMs` passed to the script.

## 3. Loop Service

- [x] 3.1 Change `LoopService.Run` signature to `Run(ctx context.Context, shardID uint, claimLimit int64, dueHandler DueHandler)` and call `ClaimDueTasksForShard(ctx, shardID, claimLimit)`.
- [x] 3.2 Update the `LoopRunner` interface in `internal/handler/zsetworker.go` to match the new `Run` signature.

## 4. Worker Runner

- [x] 4.1 Inject `*config.Config` into `ZSetWorkerRunner` (constructor + `RegisterZSetWorkerRunner`).
- [x] 4.2 In `RunZSetWorker`, read `cfg.Redis.SchedulerShards` and `cfg.Redis.SchedulerClaimLimit`, then spawn one `go r.loopService.Run(ctx, uint(shardID), int64(claimLimit), handler)` per shard.
- [x] 4.3 Remove the hard-coded `defaultClaimLimit = 50` constant (no longer used).

## 5. Tests

- [x] 5.1 Delete obsolete cross-shard tests: `TestClaimDueTasksMergesAcrossShards`, `TestClaimDueTasksEarliestNextAcrossShards`, `TestClaimDueTasksCappedAtLimit`, and the cross-shard merge assertion in `TestClaimDueTasks`.
- [x] 5.2 Adapt `TestClaimDueTasksZeroLimit` (rename to `...ForShard`) and `TestClaimDueTasksLockPreventsReclaim` to the per-shard signature.
- [x] 5.3 Add a test asserting `ClaimDueTasksForShard` claims only from its own `scheduler:queue:<id>` and ignores other shards.

## 6. Verification

- [x] 6.1 Run `go build ./...` and `go test ./...` in `ping-service` to confirm compilation and behavior.
