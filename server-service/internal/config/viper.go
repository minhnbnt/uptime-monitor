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

	defaults := map[string]any{

		"log.level": "info",

		"token.access_ttl":     "15m",
		"token.refresh_ttl":    "168h",
		"token.access_issuer":  "uptime-monitor",
		"token.refresh_issuer": "uptime-monitor-refresh",

		"argon2.memory":      16384,
		"argon2.iterations":  2,
		"argon2.parallelism": 1,
		"argon2.salt_length": 16,
		"argon2.key_length":  32,

		"temporal.host":              "localhost:7233",
		"temporal.task_queue":        "ping-task-queue",
		"temporal.workflow_name":     "ping-workflow",
		"temporal.digest_task_queue": "digest-task-queue",

		"scheduler.backend": "redis",
		"db.port":           "5432",
		"redis.db":          0,
		"grpc.port":         "50051",

		"auth.issuer": "http://gotrue:9999",

		"mail.smtp_host":     "localhost",
		"mail.smtp_port":     587,
		"mail.smtp_user":     "",
		"mail.smtp_password": "",
		"mail.from_address":  "noreply@uptime.local",

		"cors.allowed_origins":   []string{"*"},
		"cors.allowed_methods":   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		"cors.allowed_headers":   []string{"Authorization", "Content-Type"},
		"cors.max_age":           86400,
		"cors.allow_credentials": true,
	}

	for key, value := range defaults {
		v.SetDefault(key, value)
	}
}

func bindEnvVars(v *viper.Viper) error {

	envMap := map[string]string{

		"db.host":     "DB_HOST",
		"db.port":     "DB_PORT",
		"db.user":     "DB_USER",
		"db.password": "DB_PASSWORD",
		"db.dbname":   "DB_NAME",

		"redis.addr":     "REDIS_ADDR",
		"redis.password": "REDIS_PASSWORD",
		"redis.db":       "REDIS_DB",

		"jwt.key": "JWT_KEY",

		"log.level": "LOG_LEVEL",

		"temporal.host":              "TEMPORAL_HOST",
		"temporal.task_queue":        "TEMPORAL_TASK_QUEUE",
		"temporal.workflow_name":     "TEMPORAL_WORKFLOW_NAME",
		"temporal.digest_task_queue": "TEMPORAL_DIGEST_TASK_QUEUE",

		"grpc.port": "GRPC_PORT",

		"auth.issuer": "AUTH_ISSUER",

		"kubeconfig": "KUBECONFIG",

		"mail.smtp_host":     "SMTP_HOST",
		"mail.smtp_port":     "SMTP_PORT",
		"mail.smtp_user":     "SMTP_USER",
		"mail.smtp_password": "SMTP_PASSWORD",
		"mail.from_address":  "SMTP_FROM",

		"cors.allowed_origins": "CORS_ALLOWED_ORIGINS",
		"cors.allowed_methods": "CORS_ALLOWED_METHODS",
		"cors.allowed_headers": "CORS_ALLOWED_HEADERS",
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
