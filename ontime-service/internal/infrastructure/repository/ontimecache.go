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
	// v2 bumps the suffix away from the old string-format keys ("__NULL__"
	// or a bare percentage): HGETALL/HSET against those would fail with
	// WRONGTYPE until their TTL expires. Old entries die within an hour.
	ontimeKeySuffix = ":stats:v2"
	ontimeTTL       = 1 * time.Hour
	todayTTL        = 10 * time.Second
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

// RedisKey is the single source of truth for the ontime cache key layout.
// Producers and consumers (tests included) MUST build keys through it —
// hand-rolled format strings are how the v1→v2 migration broke three tests.
func RedisKey(endpointID uint, day time.Time) string {
	return fmt.Sprintf(
		"%s%d:%s%s", ontimeKeyPrefix, endpointID,
		day.Format("2006-01-02"), ontimeKeySuffix,
	)
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func (r *OntimeCacheRepository) MGet(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]dto.DayResult, error) {

	if len(keys) == 0 {
		return nil, nil
	}

	pipe := r.client.Pipeline()

	cmds := make([]*redis.MapStringStringCmd, len(keys))
	for i, k := range keys {
		cmds[i] = pipe.HGetAll(ctx, RedisKey(k.EndpointID, k.Date))
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	result := make(map[dto.BatchGetOntimeItem]dto.DayResult, len(keys))
	for i, cmd := range cmds {

		fields, err := cmd.Result()
		if err != nil || len(fields) == 0 {
			continue
		}

		stats, err := mapToDayResult(fields)
		if err != nil {
			continue
		}

		result[keys[i]] = stats
	}

	return result, nil
}

func (r *OntimeCacheRepository) MSet(ctx context.Context, items map[dto.BatchGetOntimeItem]dto.DayResult) error {

	if len(items) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for key, stats := range items {

		ttl := ontimeTTL
		if isToday(key.Date) {
			ttl = todayTTL
		}

		key := RedisKey(key.EndpointID, key.Date)
		value := map[string]string{
			"has_data": lo.If(stats.HasData, "1").Else("0"),
			"uptime":   formatFloat(stats.Uptime),
			"unknown":  formatFloat(stats.Unknown),
		}

		pipe.HSet(ctx, key, value)
		pipe.Expire(ctx, key, ttl)
	}

	_, err := pipe.Exec(ctx)

	return err
}

func mapToDayResult(fields map[string]string) (dto.DayResult, error) {

	stats := dto.DayResult{}

	if unknown, err := strconv.ParseFloat(fields["unknown"], 64); err == nil {
		stats.Unknown = unknown
	}

	if fields["has_data"] != "1" {
		return stats, nil
	}

	uptime, err := strconv.ParseFloat(fields["uptime"], 64)
	if err != nil {
		return stats, err
	}

	stats.HasData = true
	stats.Uptime = uptime

	return stats, nil
}
