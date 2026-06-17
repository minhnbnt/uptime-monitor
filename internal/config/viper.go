package config

import (
	"fmt"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"
)

func initConfig(configPath string) (*Config, error) {

	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
	)

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

	v.SetDefault("log.level", "info")

	v.SetDefault("token.access_ttl", "15m")
	v.SetDefault("token.refresh_ttl", "168h")
	v.SetDefault("token.access_issuer", "uptime-monitor")
	v.SetDefault("token.refresh_issuer", "uptime-monitor-refresh")

	v.SetDefault("argon2.memory", 16384)
	v.SetDefault("argon2.iterations", 2)
	v.SetDefault("argon2.parallelism", 1)
	v.SetDefault("argon2.salt_length", 16)
	v.SetDefault("argon2.key_length", 32)

	v.SetDefault("temporal.host", "localhost:7233")
	v.SetDefault("temporal.task_queue", "ping-task-queue")
	v.SetDefault("temporal.workflow_name", "ping-workflow")

	v.SetDefault("scheduler.backend", "redis")
	v.SetDefault("db.port", "5432")
	v.SetDefault("redis.db", 0)

	v.SetDefault("mail.smtp_host", "localhost")
	v.SetDefault("mail.smtp_port", 587)
	v.SetDefault("mail.smtp_user", "")
	v.SetDefault("mail.smtp_password", "")
	v.SetDefault("mail.from_address", "noreply@uptime.local")
}

func bindEnvVars(v *viper.Viper) error {

	envMap := map[string]string{
		"db.host":                "DB_HOST",
		"db.port":                "DB_PORT",
		"db.user":                "DB_USER",
		"db.password":            "DB_PASSWORD",
		"db.dbname":              "DB_NAME",
		"redis.addr":             "REDIS_ADDR",
		"redis.password":         "REDIS_PASSWORD",
		"redis.db":               "REDIS_DB",
		"jwt.key":                "JWT_KEY",
		"log.level":              "LOG_LEVEL",
		"temporal.host":          "TEMPORAL_HOST",
		"temporal.task_queue":    "TEMPORAL_TASK_QUEUE",
		"temporal.workflow_name": "TEMPORAL_WORKFLOW_NAME",

		"mail.smtp_host":     "SMTP_HOST",
		"mail.smtp_port":     "SMTP_PORT",
		"mail.smtp_user":     "SMTP_USER",
		"mail.smtp_password": "SMTP_PASSWORD",
		"mail.from_address":  "SMTP_FROM",
	}

	for key, env := range envMap {
		if err := v.BindEnv(key, env); err != nil {
			return fmt.Errorf("bind env var %s: %w", key, err)
		}
	}

	return nil
}

func RegisterConfig(cfg *Config) func(do.Injector) {
	return func(i do.Injector) {
		do.ProvideValue(i, cfg)
	}
}

func RegisterConfigPath(configPath string) func(do.Injector) {
	return func(i do.Injector) {
		do.Provide(i, func(i do.Injector) (*Config, error) {
			return initConfig(configPath)
		})
	}
}
