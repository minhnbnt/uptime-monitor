package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/utils"
)

const pushNextKeyPrefix = "push:next:"

// ponytail: single check-and-set gate per session; revisit only if push
// traffic ever needs per-key fairness beyond one milestone.
var pushGateScript = redis.NewScript(`
	local cur = redis.call('GET', KEYS[1])
	if cur and tonumber(ARGV[1]) < tonumber(cur) then
		return {0, cur}
	end
	redis.call('SET', KEYS[1], ARGV[2], 'PX', ARGV[3])
	return {1, ARGV[2]}
`)

type PushRateLimiter struct {
	client *redis.Client
}

func NewPushRateLimiter(client *redis.Client) *PushRateLimiter {
	return &PushRateLimiter{client: client}
}

func RegisterPushRateLimiter(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PushRateLimiter, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return NewPushRateLimiter(wrapper.GetClient()), nil
	})
}

func pushNextKey(sessionID string) string {
	return pushNextKeyPrefix + sessionID
}

func (l *PushRateLimiter) Allow(ctx context.Context, sessionID string, interval time.Duration) (time.Time, bool, error) {

	next, err := utils.NextExecutionTime(sessionID, interval)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("next execution time: %w", err)
	}

	cmd := pushGateScript.Run(
		ctx, l.client,
		[]string{pushNextKey(sessionID)},
		time.Now().UnixMilli(),
		next.UnixMilli(),
		(interval * 3).Milliseconds(),
	)

	res, err := cmd.Slice()
	if err != nil {
		return time.Time{}, false, fmt.Errorf("push gate script: %w", err)
	}

	admitted := res[0].(int64)
	return next, admitted == 1, nil
}

func (l *PushRateLimiter) Release(ctx context.Context, sessionID string) error {
	return l.client.Del(ctx, pushNextKey(sessionID)).Err()
}
