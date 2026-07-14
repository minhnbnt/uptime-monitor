package config

import (
	"fmt"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"
)

func InitConfig(configPath string) (*Config, error) {
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
		"server.port":      "8084",
		"grpc.port":        "50052",
		"grpc.server_addr": "server:50051",
		"log.level":        "info",
		"db.port":          "5432",
		"redis.db":         0,
	}

	for key, value := range defaults {
		v.SetDefault(key, value)
	}
}

func bindEnvVars(v *viper.Viper) error {
	envMap := map[string]string{
		"server.port":      "PORT",
		"grpc.port":        "GRPC_PORT",
		"grpc.server_addr": "GRPC_SERVER_ADDR",
		"db.host":          "DB_HOST",
		"db.port":          "DB_PORT",
		"db.user":          "DB_USER",
		"db.password":      "DB_PASSWORD",
		"db.dbname":        "DB_NAME",
		"redis.addr":       "REDIS_ADDR",
		"redis.db":         "REDIS_DB",
		"log.level":        "LOG_LEVEL",
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
			return InitConfig(configPath)
		})
	}
}
