package repository

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

const schedulerZSetKey = "scheduler:queue"

func schedulerMetaKey(id uint) string {
	return fmt.Sprintf("scheduler:meta:%d", id)
}

type ZSetSchedulerRepository struct {
	client *redis.Client
}

func RegisterZSetSchedulerRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ZSetSchedulerRepository, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return &ZSetSchedulerRepository{client: wrapper.GetClient()}, nil
	})
}

func (r *ZSetSchedulerRepository) Register(ctx context.Context, endpoint *domain.Endpoint) error {

	idStr := fmt.Sprintf("%d", endpoint.ID)
	offset := utils.GenerateOffset(idStr, endpoint.Interval)
	score := time.Now().UnixMilli() + offset.Milliseconds()

	pipe := r.client.Pipeline()

	member := strconv.FormatUint(uint64(endpoint.ID), 10)
	pipe.ZAdd(ctx, schedulerZSetKey, redis.Z{
		Score:  float64(score),
		Member: member,
	})

	pipe.HSet(ctx, schedulerMetaKey(endpoint.ID),
		"url", endpoint.URL,
		"method", endpoint.Method,
		"expected_code", strconv.Itoa(endpoint.ExpectedCode),
		"interval_ns", strconv.FormatInt(int64(endpoint.Interval), 10),
	)

	_, err := pipe.Exec(ctx)
	return err
}
