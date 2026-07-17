package config

import (
	"fmt"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"
)

func initConfig(configPath string) (*Config, error) {

	v := viper.NewWithOptions(viper.KeyDelimiter("."))

	setDefaults(v)

	if err := bindEnvVars(v); err != nil {
		return nil, fmt.Errorf("bind env vars: %w", err)
	}

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	if configPath != "" {
		v.SetConfigFile(configPath)
	}

	if err := v.ReadInConfig(); err != nil {
		if configPath != "" {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	cfg := new(Config)
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {

	defaults := map[string]any{
		"log.level":                "info",
		"redis.db":                 0,
		"redis.scheduler_shards":   1,
		"server.port":              "8083",
		"grpc.server_addr":         "server:50051",
		"grpc.event_addr":          "ontime:50052",
	}

	for key, value := range defaults {
		v.SetDefault(key, value)
	}
}

func bindEnvVars(v *viper.Viper) error {

	envMap := map[string]string{
		"redis.addr":             "REDIS_ADDR",
		"redis.password":         "REDIS_PASSWORD",
		"redis.db":               "REDIS_DB",
		"redis.scheduler_shards": "REDIS_SCHEDULER_SHARDS",

		"server.port": "PING_SERVICE_PORT",

		"grpc.server_addr": "GRPC_SERVER_ADDR",
		"grpc.event_addr":  "GRPC_EVENT_ADDR",

		"log.level": "LOG_LEVEL",
	}

	for key, env := range envMap {
		if err := v.BindEnv(key, env); err != nil {
			return fmt.Errorf("bind env var %s: %w", key, err)
		}
	}

	return nil
}

func RegisterConfig(cfg *Config) func(do.Injector) {
	return func(i do.Injector) { do.ProvideValue(i, cfg) }
}

func RegisterConfigPath(configPath string) func(do.Injector) {
	return func(i do.Injector) {
		do.Provide(i, func(i do.Injector) (*Config, error) {
			return initConfig(configPath)
		})
	}
}
