package config

import (
	"fmt"
	"os"
	"strconv"

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

func newRedisConfig(i do.Injector) (*redis.Options, error) {

	dbIndex, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	return &redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       dbIndex,
	}, nil
}

func newRedisClientWrapper(i do.Injector) (*RedisClientWrapper, error) {

	config := do.MustInvoke[*redis.Options](i)
	client := redis.NewClient(config)

	return &RedisClientWrapper{client: client}, nil
}

func RegisterRedisClient(i do.Injector) {
	do.Provide(i, newRedisClientWrapper)
	do.Provide(i, newRedisConfig)
}
