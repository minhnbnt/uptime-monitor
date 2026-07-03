package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

const (
	schedulerQueueKey = "scheduler:queue"
	claimLock         = 10 * time.Second
)

type ScheduledTask struct {
	EndpointID uint
	Score      int64 // next execution time in UnixMilliseconds
}

type ZSetScheduleRepository struct {
	client *redis.Client
}

func NewZSetScheduleRepository(client *redis.Client) *ZSetScheduleRepository {
	return &ZSetScheduleRepository{client: client}
}

func RegisterZSetScheduleRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ZSetScheduleRepository, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return NewZSetScheduleRepository(wrapper.GetClient()), nil
	})
}

func (r *ZSetScheduleRepository) Register(ctx context.Context, endpoint *domain.Endpoint) error {

	idStr := fmt.Sprint(endpoint.ID)
	offset := utils.GenerateOffset(idStr, endpoint.Interval)
	score := time.Now().UnixMilli() + offset.Milliseconds()

	cmd := r.client.ZAdd(ctx, schedulerQueueKey, redis.Z{
		Score:  float64(score),
		Member: idStr,
	})

	return cmd.Err()
}

func (r *ZSetScheduleRepository) RegisterBatch(ctx context.Context, endpoints []domain.Endpoint) error {

	pipe := r.client.Pipeline()

	for _, ep := range endpoints {
		idStr := fmt.Sprint(ep.ID)
		offset := utils.GenerateOffset(idStr, ep.Interval)
		score := time.Now().UnixMilli() + offset.Milliseconds()
		pipe.ZAdd(ctx, schedulerQueueKey, redis.Z{
			Score:  float64(score),
			Member: idStr,
		})
	}

	_, err := pipe.Exec(ctx)

	return err
}

func (r *ZSetScheduleRepository) Unregister(ctx context.Context, endpointID uint) error {

	pipe := r.client.Pipeline()

	pipe.ZRem(ctx, schedulerQueueKey, fmt.Sprint(endpointID))
	pipe.Del(ctx, metaCacheKey(endpointID))

	_, err := pipe.Exec(ctx)

	return err
}

func (r *ZSetScheduleRepository) ClaimDueTasks(
	ctx context.Context, limit int64,
) (due []ScheduledTask, next ScheduledTask, hasNext bool, err error) {

	if limit <= 0 {
		return nil, ScheduledTask{}, false, nil
	}

	cmd := claimScript.Run(
		ctx, r.client, []string{schedulerQueueKey},
		fmt.Sprint(time.Now().UnixMilli()),
		fmt.Sprint(limit),
		fmt.Sprint(claimLock.Milliseconds()),
	)

	return collectScheduledTask(cmd)
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
