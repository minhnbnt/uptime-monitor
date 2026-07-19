package recorder

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
)

const (
	statusKey = "endpoint:status"
	statusTTL = 7 * 24 * time.Hour
)

type DedupRecorder struct {
	db     *gorm.DB
	rdb    *redis.Client
	logger *slog.Logger
}

func RegisterDedupRecorder(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*DedupRecorder, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		redisWrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		logger := do.MustInvoke[*slog.Logger](i)
		return &DedupRecorder{
			db:     dbWrapper.GetDB(),
			rdb:    redisWrapper.GetClient(),
			logger: logger,
		}, nil
	})
}

func (r *DedupRecorder) RecordEvent(ctx context.Context, endpointID uint, status domain.ServerStatus) error {

	lastStatus, err := r.rdb.HGet(ctx, statusKey, fmt.Sprint(endpointID)).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("get status from redis: %w", err)
	}

	if domain.ServerStatus(lastStatus) == status {
		return nil
	}

	event := &domain.ServerEvent{
		ID:         uuid.New(),
		EndpointID: endpointID,
		Status:     status,
		Time:       time.Now(),
	}

	result := r.db.WithContext(ctx).Create(event)
	if err := result.Error; err != nil {
		return fmt.Errorf("save event: %w", err)
	}

	cmd := r.rdb.HSet(ctx, statusKey, fmt.Sprint(endpointID), string(status))
	if err := cmd.Err(); err != nil {
		return fmt.Errorf("set status in redis: %w", err)
	}

	r.rdb.HExpire(ctx, statusKey, statusTTL, fmt.Sprint(endpointID))

	return nil
}
