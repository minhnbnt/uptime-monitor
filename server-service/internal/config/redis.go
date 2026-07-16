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

func newRedisConfig(cfg *Config) (*redis.Options, error) {
	return &redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}, nil
}

func newRedisClientWrapper(i do.Injector) (*RedisClientWrapper, error) {

	config := do.MustInvoke[*redis.Options](i)
	client := redis.NewClient(config)

	return &RedisClientWrapper{client: client}, nil
}

func RegisterRedisClient(i do.Injector) {

	do.Provide(i, func(i do.Injector) (*redis.Options, error) {
		cfg := do.MustInvoke[*Config](i)
		return newRedisConfig(cfg)
	})

	do.Provide(i, newRedisClientWrapper)
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}
