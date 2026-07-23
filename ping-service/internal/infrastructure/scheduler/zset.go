package scheduler

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

const (
	schedulerQueuePrefix = "scheduler:queue"
	claimLock            = 10 * time.Second
)

func schedulerShardKey(shardCount int, endpointID uint) string {

	if shardCount < 1 {
		shardCount = 1
	}

	hasher := fnv.New32a()
	fmt.Fprint(hasher, endpointID)

	shardID := hasher.Sum32() % uint32(shardCount)

	return shardKey(uint(shardID))
}

func shardKey(shardID uint) string {
	return fmt.Sprintf("%s:%d", schedulerQueuePrefix, shardID)
}

func GenerateOffset(id any, interval time.Duration) time.Duration {

	hasher := fnv.New64a()
	fmt.Fprint(hasher, id)

	offset := hasher.Sum64() % uint64(interval)
	return time.Duration(offset)
}

type ScheduledTask struct {
	EndpointID uint
	Score      int64 // next execution time in UnixMilliseconds
}

type ZSetScheduleRepository struct {
	client     *redis.Client
	shardCount int
}

func NewZSetScheduleRepository(client *redis.Client, shardCount int) *ZSetScheduleRepository {

	if shardCount < 1 {
		shardCount = 1
	}

	return &ZSetScheduleRepository{
		shardCount: shardCount,
		client:     client,
	}
}

func RegisterZSetScheduleRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ZSetScheduleRepository, error) {

		cfg := do.MustInvoke[*config.Config](i)
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)

		return NewZSetScheduleRepository(
			wrapper.GetClient(),
			cfg.Redis.SchedulerShards,
		), nil
	})
}

func (r *ZSetScheduleRepository) Register(
	ctx context.Context,
	endpoint *domain.Endpoint,
) error {
	return r.RegisterBatch(ctx, []domain.Endpoint{*endpoint})
}

func (r *ZSetScheduleRepository) RegisterBatch(
	ctx context.Context,
	endpoints []domain.Endpoint,
) error {

	if len(endpoints) == 0 {
		return nil
	}

	groups := make(map[string][]redis.Z)
	for _, endpoint := range endpoints {

		id := fmt.Sprint(endpoint.ID)
		key := schedulerShardKey(r.shardCount, endpoint.ID)
		offset := GenerateOffset(id, endpoint.Interval)

		member := redis.Z{
			Member: fmt.Sprint(id),
			Score:  float64(offset.Milliseconds()),
		}

		groups[key] = append(groups[key], member)
	}

	pipeliner := r.client.Pipeline()
	for key, members := range groups {
		pipeliner.ZAdd(ctx, key, members...)
	}

	_, err := pipeliner.Exec(ctx)
	return err
}

func (r *ZSetScheduleRepository) Unregister(ctx context.Context, endpointID uint) error {

	zsetKey := schedulerShardKey(r.shardCount, endpointID)
	cmd := r.client.ZRem(ctx, zsetKey, fmt.Sprint(endpointID))

	return cmd.Err()
}

func (r *ZSetScheduleRepository) MoveIfWrongShard(
	ctx context.Context, shardID uint, due []ScheduledTask,
) ([]ScheduledTask, error) {

	filtered := make([]ScheduledTask, 0, len(due))
	claimedKey := shardKey(shardID)

	for _, task := range due {

		correctKey := schedulerShardKey(r.shardCount, task.EndpointID)
		if correctKey == claimedKey {
			filtered = append(filtered, task)
			continue
		}

		_, err := r.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {

			pipe.ZAdd(ctx, correctKey, redis.Z{
				Member: fmt.Sprint(task.EndpointID),
				Score:  float64(time.Now().UnixMilli()),
			})

			pipe.ZRem(ctx, claimedKey, fmt.Sprint(task.EndpointID))

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("move task %d to correct shard: %w", task.EndpointID, err)
		}
	}

	return filtered, nil
}

func (r *ZSetScheduleRepository) ClaimDueTasksForShard(
	ctx context.Context, shardID uint, limit int64,
) (due []ScheduledTask, next ScheduledTask, hasNext bool, err error) {

	if limit <= 0 {
		return nil, ScheduledTask{}, false, nil
	}

	now := time.Now().UnixMilli()
	lockMs := claimLock.Milliseconds()

	cmd := claimScript.Run(
		ctx, r.client,
		[]string{shardKey(shardID)},
		fmt.Sprint(now),
		fmt.Sprint(limit),
		fmt.Sprint(lockMs),
	)

	return collectScheduledTask(cmd)
}

// claimScript atomically claims at most N due tasks and peeks the next future task.
//
// KEYS[1] = scheduler:queue:<shardID>
// ARGV[1] = now in UnixMilliseconds
// ARGV[2] = max number of due tasks to claim
// ARGV[3] = claim lock duration in milliseconds
// Returns: {due_array, next_array}
//
//	due_array:  [member1, score1, member2, score2, ...] — scores bumped to now+lockMs
//	next_array: [member, score] — stays in ZSET, or [] if none
var claimScript = redis.NewScript(`
	local due = redis.call("ZRANGEBYSCORE", KEYS[1], "-inf", ARGV[1], "WITHSCORES", "LIMIT", "0", ARGV[2])
	local next = redis.call("ZRANGEBYSCORE", KEYS[1], "(" .. ARGV[1], "+inf", "WITHSCORES", "LIMIT", "0", "1")

	local lockScore = tonumber(ARGV[1]) + tonumber(ARGV[3])

	local zaddArgs = {KEYS[1]}
	for i = 1, #due, 2 do
		table.insert(zaddArgs, lockScore)
		table.insert(zaddArgs, due[i])
	end

	if #zaddArgs > 1 then
		redis.call("ZADD", unpack(zaddArgs))
	end

	return {due, next}
`)

func collectScheduledTask(cmd *redis.Cmd) (due []ScheduledTask, next ScheduledTask, hasNext bool, err error) {

	result, err := cmd.Result()
	if err != nil {
		return nil, ScheduledTask{}, false, fmt.Errorf("claim due tasks: %w", err)
	}

	vals, ok := result.([]any)
	if !ok {
		return nil, ScheduledTask{}, false, fmt.Errorf("unexpected script result type: %T", result)
	}

	if len(vals) != 2 {
		return nil, ScheduledTask{}, false, fmt.Errorf("unexpected script result length: %d", len(vals))
	}

	dueRaw, ok := vals[0].([]any)
	if !ok {
		return nil, ScheduledTask{}, false, fmt.Errorf("invalid dueRaw type: %T", vals[0])
	}

	nextRaw, ok := vals[1].([]any)
	if !ok {
		return nil, ScheduledTask{}, false, fmt.Errorf("invalid nextRaw type: %T", vals[1])
	}

	for i := 0; i+1 < len(dueRaw); i += 2 {

		task, err := getScheduledTask(dueRaw[i], dueRaw[i+1])
		if err != nil {
			return nil, ScheduledTask{}, false, fmt.Errorf("parse due task at index %d: %w", i, err)
		}

		due = append(due, *task)
	}

	if len(nextRaw) >= 2 {

		task, err := getScheduledTask(nextRaw[0], nextRaw[1])
		if err != nil {
			return due, ScheduledTask{}, false, fmt.Errorf("parse next task: %w", err)
		}

		hasNext = true
		next = *task
	}

	return due, next, hasNext, nil
}

func getScheduledTask(member, scoreStr any) (*ScheduledTask, error) {

	memberStr, ok := member.(string)
	if !ok {
		return nil, fmt.Errorf("invalid member type: %T", member)
	}

	scoreStrStr, ok := scoreStr.(string)
	if !ok {
		return nil, fmt.Errorf("invalid score type: %T", scoreStr)
	}

	id, err := strconv.ParseUint(memberStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint id: %w", err)
	}

	score, err := strconv.ParseInt(scoreStrStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse score: %w", err)
	}

	return &ScheduledTask{
		EndpointID: uint(id),
		Score:      score,
	}, nil
}
