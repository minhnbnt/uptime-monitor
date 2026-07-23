## 1. Scheduler — MoveIfWrongShard

- [x] 1.1 Add `MoveIfWrongShard(ctx, shardID uint, due []ScheduledTask) ([]ScheduledTask, error)` method to `ZSetScheduleRepository`
- [x] 1.2 For each task: compare `schedulerShardKey(endpointID)` vs `shardKey(shardID)` — if same, keep in batch
- [x] 1.3 If mismatched: atomic `TxPipelined` with `ZADD` to correct shard (score = `time.Now().UnixMilli()`) + `ZREM` from wrong shard
- [x] 1.4 Return filtered batch (only tasks already in correct shard)

## 2. ZsetLoopService — integrate MoveIfWrongShard

- [x] 2.1 In `Run`, after `ClaimDueTasksForShard`, call `schedulerStorage.MoveIfWrongShard(ctx, shardID, due)`
- [x] 2.2 Pass the filtered batch to `runIteration`
- [x] 2.3 Remove `scoreUpdater` field and `calculateNextScore` from `ZsetLoopService`

## 3. PingLoopService — nextExecutionTime

- [x] 3.1 Add `GenerateOffset` call (import from `scheduler` package)
- [x] 3.2 Implement `nextExecutionTime(endpointID uint, interval time.Duration) int64`
- [x] 3.3 Remove `calculateNextScore`
- [x] 3.4 Update `pingAndRecordEndpoint` — remove `score` param, compute next score inline via `nextExecutionTime`
- [x] 3.5 Remove `scoreUpdater` field (moved to PingLoopService)
- [x] 3.6 Add `scoreUpdater` field back to `PingLoopService` (keep the Update call after Record)

## 4. Remove PingTask — revert channel to *domain.Endpoint

- [x] 4.1 Delete `service/pingtask.go`
- [x] 4.2 Revert `PingService` interface in `handler/zsetworker.go` to `Run(ctx, <-chan *domain.Endpoint)`
- [x] 4.3 Change channel in `zsetworker.go` back to `chan *domain.Endpoint`
- [x] 4.4 Update handler to send `*domain.Endpoint` directly
- [x] 4.5 Update `ZsetLoopService.runIteration` to build `iter.Seq[*domain.Endpoint]` instead of `*PingTask`
- [x] 4.6 Revert `DueHandler` type back to `func(ctx, iter.Seq[*domain.Endpoint])`

## 5. Update tests

- [x] 5.1 Update `zsetloop_test.go` — handler type change, no score updater
- [x] 5.2 Update `pingloop_test.go` — `nextExecutionTime` tests, remove score param from calls
- [x] 5.3 Update `mocks_test.go` — remove duplicate mocks

## 6. Verify

- [x] 6.1 `go build ./...` passes
- [x] 6.2 `go test ./...` passes
- [x] 6.3 `golangci-lint run ./...` passes
