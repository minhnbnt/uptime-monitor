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

const schedulerQueueKey = "scheduler:queue"

type ScheduledTask struct {
	EndpointID uint
	Score      int64 // next execution time in UnixMilliseconds
}

type ZSetScheduleRepository struct {
	client *redis.Client
}

func RegisterZSetScheduleRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ZSetScheduleRepository, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return &ZSetScheduleRepository{client: wrapper.GetClient()}, nil
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

func (r *ZSetScheduleRepository) Unregister(ctx context.Context, endpointID uint) error {

	pipe := r.client.Pipeline()

	pipe.ZRem(ctx, schedulerQueueKey, fmt.Sprint(endpointID))
	pipe.Del(ctx, metaCacheKey(endpointID))

	_, err := pipe.Exec(ctx)

	return err
}

// claimScript atomically claims at most N due tasks and peeks the next future task.
//
// KEYS[1] = scheduler:queue
// ARGV[1] = now in UnixMilliseconds
// ARGV[2] = max number of due tasks to claim
// Returns: {due_array, next_array}
//
//	due_array:  [member1, score1, member2, score2, ...] — atomically removed from ZSET
//	next_array: [member, score] — stays in ZSET, or [] if none
var claimScript = redis.NewScript(`
	local due = redis.call("ZRANGEBYSCORE", KEYS[1], "-inf", ARGV[1], "WITHSCORES", "LIMIT", "0", ARGV[2])
	local next = redis.call("ZRANGEBYSCORE", KEYS[1], "(" .. ARGV[1], "+inf", "WITHSCORES", "LIMIT", "0", "1")
	if #due > 0 then
		local members = {}
		for i = 1, #due, 2 do
			members[#members + 1] = due[i]
		end
		redis.call("ZREM", KEYS[1], unpack(members))
	end
	return {due, next}
`)

func collectRawValues(cmd *redis.Cmd) (dueRaw, nextRaw []any, err error) {

	result, err := cmd.Result()
	if err != nil {
		return nil, nil, fmt.Errorf("claim due tasks: %w", err)
	}

	vals, ok := result.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("unexpected script result type: %T", result)
	}

	if len(vals) != 2 {
		return nil, nil, fmt.Errorf("unexpected script result length: %d", len(vals))
	}

	dueRaw, ok = vals[0].([]any)
	if !ok {
		return nil, nil, fmt.Errorf("invalid dueRaw type: %T", vals[0])
	}

	nextRaw, ok = vals[1].([]any)
	if !ok {
		return nil, nil, fmt.Errorf("invalid nextRaw type: %T", vals[1])
	}

	return dueRaw, nextRaw, nil
}

func (r *ZSetScheduleRepository) ClaimDueTasks(
	ctx context.Context, limit int64,
) (due []ScheduledTask, next *ScheduledTask, err error) {

	if limit <= 0 {
		return nil, nil, nil
	}

	now := time.Now()

	cmd := claimScript.Run(
		ctx, r.client, []string{schedulerQueueKey},
		fmt.Sprint(now.UnixMilli()),
		fmt.Sprint(limit),
	)

	dueRaw, nextRaw, err := collectRawValues(cmd)
	if err != nil {
		return nil, nil, err
	}

	for i := 0; i+1 < len(dueRaw); i += 2 {

		task, err := getScheduledTask(dueRaw[i], dueRaw[i+1])
		if err != nil {
			return nil, nil, fmt.Errorf("parse due task at index %d: %w", i, err)
		}

		due = append(due, *task)
	}

	if len(nextRaw) >= 2 {

		task, err := getScheduledTask(nextRaw[0], nextRaw[1])
		if err != nil {
			return due, nil, fmt.Errorf("parse next task: %w", err)
		}

		next = task
	}

	return due, next, nil
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
