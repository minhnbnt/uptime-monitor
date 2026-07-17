package scheduler

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/utils"
)

const (
	schedulerQueuePrefix = "scheduler:queue"
	claimLock            = 10 * time.Second
)

func schedulerShardKey(shardCount int, endpointID uint) string {
	if shardCount <= 1 {
		return schedulerQueuePrefix
	}
	h := fnv.New32a()
	h.Write([]byte(strconv.FormatUint(uint64(endpointID), 10)))
	shardID := h.Sum32() % uint32(shardCount)
	return fmt.Sprintf("%s:%d", schedulerQueuePrefix, shardID)
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
	return &ZSetScheduleRepository{client: client, shardCount: shardCount}
}

func RegisterZSetScheduleRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ZSetScheduleRepository, error) {
		cfg := do.MustInvoke[*config.Config](i)
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return NewZSetScheduleRepository(wrapper.GetClient(), cfg.Redis.SchedulerShards), nil
	})
}

func (r *ZSetScheduleRepository) Register(ctx context.Context, endpoint *domain.Endpoint) error {
	return r.RegisterBatch(ctx, []domain.Endpoint{*endpoint})
}

func (r *ZSetScheduleRepository) RegisterBatch(ctx context.Context, endpoints []domain.Endpoint) error {
	if len(endpoints) == 0 {
		return nil
	}

	if r.shardCount <= 1 {
		members := lo.Map(endpoints, func(endpoint domain.Endpoint, _ int) redis.Z {
			idStr := fmt.Sprint(endpoint.ID)
			offset := utils.GenerateOffset(idStr, endpoint.Interval)
			score := offset.Milliseconds()
			return redis.Z{Member: idStr, Score: float64(score)}
		})
		_, err := r.client.ZAdd(ctx, schedulerQueuePrefix, members...).Result()
		return err
	}

	type entry struct {
		key    string
		member redis.Z
	}
	entries := lo.Map(endpoints, func(endpoint domain.Endpoint, _ int) entry {
		idStr := fmt.Sprint(endpoint.ID)
		offset := utils.GenerateOffset(idStr, endpoint.Interval)
		return entry{
			key:    schedulerShardKey(r.shardCount, endpoint.ID),
			member: redis.Z{Member: idStr, Score: float64(offset.Milliseconds())},
		}
	})

	groups := make(map[string][]redis.Z)
	for _, e := range entries {
		groups[e.key] = append(groups[e.key], e.member)
	}
	for key, members := range groups {
		if _, err := r.client.ZAdd(ctx, key, members...).Result(); err != nil {
			return err
		}
	}
	return nil
}

func (r *ZSetScheduleRepository) Unregister(ctx context.Context, endpointID uint) error {
	return r.client.ZRem(ctx, schedulerShardKey(r.shardCount, endpointID), fmt.Sprint(endpointID)).Err()
}

func (r *ZSetScheduleRepository) ClaimDueTasks(
	ctx context.Context, limit int64,
) (due []ScheduledTask, next ScheduledTask, hasNext bool, err error) {
	if limit <= 0 {
		return nil, ScheduledTask{}, false, nil
	}

	if r.shardCount <= 1 {
		cmd := claimScript.Run(
			ctx, r.client, []string{schedulerQueuePrefix},
			fmt.Sprint(time.Now().UnixMilli()),
			fmt.Sprint(limit),
			fmt.Sprint(claimLock.Milliseconds()),
		)
		return collectScheduledTask(cmd)
	}

	now := time.Now().UnixMilli()
	perShardLimit := limit/int64(r.shardCount) + 1
	lockMs := claimLock.Milliseconds()

	type shardResult struct {
		due     []ScheduledTask
		next    ScheduledTask
		hasNext bool
		err     error
	}
	results := make([]shardResult, r.shardCount)
	var wg sync.WaitGroup
	for i := range r.shardCount {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("%s:%d", schedulerQueuePrefix, idx)
			cmd := claimScript.Run(ctx, r.client, []string{key},
				fmt.Sprint(now), fmt.Sprint(perShardLimit), fmt.Sprint(lockMs))
			d, n, h, e := collectScheduledTask(cmd)
			results[idx] = shardResult{due: d, next: n, hasNext: h, err: e}
		}(i)
	}
	wg.Wait()

	var allDue []ScheduledTask
	earliest := ScheduledTask{}
	hasAnyNext := false
	for _, res := range results {
		if res.err != nil {
			return nil, ScheduledTask{}, false, res.err
		}
		allDue = append(allDue, res.due...)
		if res.hasNext && (!hasAnyNext || res.next.Score < earliest.Score) {
			earliest = res.next
			hasAnyNext = true
		}
	}

	if int64(len(allDue)) > limit {
		allDue = allDue[:limit]
	}

	return allDue, earliest, hasAnyNext, nil
}

// claimScript atomically claims at most N due tasks and peeks the next future task.
//
// KEYS[1] = scheduler:queue
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
