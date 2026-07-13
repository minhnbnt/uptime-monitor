package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/jwt"
)

const revokedPrefix = "auth:revokedtoken"

type RedisRevokedTokenRepository struct {
	client *redis.Client
}

func RegisterRedisRevokedTokenRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*RedisRevokedTokenRepository, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return &RedisRevokedTokenRepository{client: wrapper.GetClient()}, nil
	})
}

func (r *RedisRevokedTokenRepository) Revoke(ctx context.Context, token *jwt.Token) error {

	jti, err := token.JTI()
	if err != nil {
		return err
	}

	expiry, err := token.Expiry()
	if err != nil {
		return err
	}

	ttl := time.Until(expiry)
	if ttl <= 0 {
		return nil
	}

	pipe := r.client.Pipeline()

	pipe.HSet(ctx, revokedPrefix, jti, ttl)
	pipe.HExpire(ctx, revokedPrefix, ttl, jti)

	_, err = pipe.Exec(ctx)

	return err
}

func (r *RedisRevokedTokenRepository) IsRevoked(ctx context.Context, jti string) (bool, error) {

	n, err := r.client.HExists(ctx, revokedPrefix, jti).Result()
	if err != nil {
		return false, err
	}

	return n, nil
}
