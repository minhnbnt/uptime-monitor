package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
)

const (
	ontimeKeyPrefix = "ontime:"
	ontimeKeySuffix = ":stats"
	ontimeTTL       = 1 * time.Hour
	todayTTL        = 10 * time.Second

	// noDataValue is the cached sentinel for "no known state for this day
	// yet" — deliberately not a valid float, so it can never be confused
	// with a real uptime percentage such as 0.00.
	noDataValue = "-"
)

func isToday(t time.Time) bool {
	now := time.Now()
	today := utils.TruncateDay(now)
	return utils.TruncateDay(t).Equal(today)
}

type OntimeCacheRepository struct {
	client *redis.Client
}

func NewOntimeCacheRepository(client *redis.Client) *OntimeCacheRepository {
	return &OntimeCacheRepository{client: client}
}

func RegisterOntimeCacheRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeCacheRepository, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return &OntimeCacheRepository{client: wrapper.GetClient()}, nil
	})
}

func redisKey(serverID uint, day time.Time) string {
	return fmt.Sprintf(
		"%s%d:%s%s", ontimeKeyPrefix, serverID,
		day.Format("2006-01-02"), ontimeKeySuffix,
	)
}

func encode(r dto.DayResult) string {
	if !r.HasData {
		return noDataValue
	}
	return fmt.Sprintf("%.2f", r.Uptime)
}

// decode is the inverse of encode. A value that's neither the sentinel nor
// a parseable float is treated as a cache miss (ok=false), never guessed at
// — a corrupted or foreign value must never be interpreted as a real uptime
// number.
func decode(raw string) (r dto.DayResult, ok bool) {
	if raw == noDataValue {
		return dto.DayResult{HasData: false}, true
	}

	uptime, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return dto.DayResult{}, false
	}

	return dto.DayResult{HasData: true, Uptime: uptime}, true
}

func (r *OntimeCacheRepository) MGet(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]dto.DayResult, error) {

	if len(keys) == 0 {
		return nil, nil
	}

	redisKeys := lo.Map(keys, func(k dto.BatchGetOntimeItem, _ int) string {
		return redisKey(k.ServerID, k.Date)
	})

	values, err := r.client.MGet(ctx, redisKeys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[dto.BatchGetOntimeItem]dto.DayResult, len(keys))
	for i, val := range values {

		if val == nil {
			continue
		}

		str, ok := val.(string)
		if !ok {
			continue
		}

		decoded, ok := decode(str)
		if !ok {
			continue
		}

		result[keys[i]] = decoded
	}

	return result, nil
}

func (r *OntimeCacheRepository) MSet(ctx context.Context, items map[dto.BatchGetOntimeItem]dto.DayResult) error {

	if len(items) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for key, result := range items {

		ttl := ontimeTTL
		if isToday(key.Date) {
			ttl = todayTTL
		}

		pipe.Set(ctx, redisKey(key.ServerID, key.Date), encode(result), ttl)
	}

	_, err := pipe.Exec(ctx)

	return err
}
