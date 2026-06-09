package revokedtoken

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
)

const revokedPrefix = "revoked_token:"

type RedisRevokedTokenRepository struct {
	client *redis.Client
}

func RegisterRedisRevokedTokenRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*RedisRevokedTokenRepository, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return &RedisRevokedTokenRepository{client: wrapper.GetClient()}, nil
	})
}

func (r *RedisRevokedTokenRepository) Revoke(ctx context.Context, token *jwtutil.Token) error {

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

	return r.client.Set(ctx, revokedPrefix+jti, time.Now().UnixMilli(), ttl).Err()
}

func (r *RedisRevokedTokenRepository) IsRevoked(ctx context.Context, jti string) (bool, error) {

	n, err := r.client.Exists(ctx, revokedPrefix+jti).Result()
	if err != nil {
		return false, err
	}

	return n > 0, nil
}
