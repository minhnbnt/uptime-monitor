package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

const (
	ontimeKeyPrefix = "ontime:"
	ontimeKeySuffix = ":stats"
	ontimeTTL       = 1 * time.Hour
	todayTTL        = 10 * time.Second
)

func isToday(t time.Time) bool {

	now := time.Now()
	today := utils.TruncateDay(now)

	return utils.TruncateDay(t).Equal(today)
}

type OntimeCacheKey struct {
	ServerID uint
	Day      time.Time
}

type OntimeCacheRepository struct {
	client *redis.Client
}

func RegisterOntimeCacheRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeCacheRepository, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return &OntimeCacheRepository{client: wrapper.GetClient()}, nil
	})
}

func ontimeCacheKey(serverID uint, day time.Time) string {
	return fmt.Sprintf(
		"%s%d:%s%s", ontimeKeyPrefix, serverID,
		day.Format("2006-01-02"), ontimeKeySuffix,
	)
}

func (r *OntimeCacheRepository) MGet(ctx context.Context, keys []OntimeCacheKey) (map[OntimeCacheKey]float64, error) {

	if len(keys) == 0 {
		return nil, nil
	}

	redisKeys := lo.Map(keys, func(k OntimeCacheKey, _ int) string {
		return ontimeCacheKey(k.ServerID, k.Day)
	})

	values, err := r.client.MGet(ctx, redisKeys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[OntimeCacheKey]float64, len(keys))
	for i, val := range values {

		if val == nil {
			continue
		}

		str, ok := val.(string)
		if !ok {
			continue
		}

		ontimePercent, err := strconv.ParseFloat(str, 64)
		if err != nil {
			continue
		}

		result[keys[i]] = ontimePercent
	}

	return result, nil
}

func (r *OntimeCacheRepository) MSet(ctx context.Context, items map[OntimeCacheKey]float64) error {

	if len(items) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for key, stats := range items {
		ttl := ontimeTTL
		if isToday(key.Day) {
			ttl = todayTTL
		}
		pipe.Set(
			ctx, ontimeCacheKey(key.ServerID, key.Day),
			strconv.FormatFloat(stats, 'f', 2, 64), ttl,
		)
	}

	_, err := pipe.Exec(ctx)

	return err
}
