package config

import (
	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"
)

type RedisClientWrapper struct {
	client *redis.Client
}

func (r *RedisClientWrapper) GetClient() *redis.Client {
	return r.client
}

func (r *RedisClientWrapper) Shutdown() error {
	return r.client.Close()
}

type RedisConfig struct {
	Addr                string `mapstructure:"addr"`
	Password            string `mapstructure:"password"`
	DB                  int    `mapstructure:"db"`
	SchedulerShards     int    `mapstructure:"scheduler_shards"`
	SchedulerClaimLimit int    `mapstructure:"scheduler_claim_limit"`
}

func RegisterRedisClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*redis.Options, error) {
		cfg := do.MustInvoke[*Config](i)
		return &redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		}, nil
	})

	do.Provide(i, func(i do.Injector) (*RedisClientWrapper, error) {
		opts := do.MustInvoke[*redis.Options](i)
		return &RedisClientWrapper{client: redis.NewClient(opts)}, nil
	})
}
