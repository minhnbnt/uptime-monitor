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

		"temporal.host":              "localhost:7233",
		"temporal.workflow_name":     "send-report",
		"temporal.digest_task_queue": "digest-task-queue",

		"server.port": "8085",

		"mail.smtp_host":                "localhost",
		"mail.smtp_port":                1025,
		"mail.smtp_user":                "",
		"mail.smtp_password":            "",
		"mail.from_address":             "noreply@uptime-monitor.local",
		"mail.disable_security":         false,
		"mail.tls_insecure_skip_verify": false,

		"auth_service.addr": "http://localhost:8081",

		"grpc.server_addr": "localhost:50051",
		"grpc.event_addr":  "localhost:50052",

		"digest.max_records": 10000,
	}

	for key, value := range defaults {
		v.SetDefault(key, value)
	}
}

func bindEnvVars(v *viper.Viper) error {

	envMap := map[string]string{

		"log.level": "LOG_LEVEL",

		"temporal.host":              "TEMPORAL_HOST",
		"temporal.workflow_name":     "TEMPORAL_WORKFLOW_NAME",
		"temporal.digest_task_queue": "TEMPORAL_DIGEST_TASK_QUEUE",

		"mail.smtp_host":                "SMTP_HOST",
		"mail.smtp_port":                "SMTP_PORT",
		"mail.smtp_user":                "SMTP_USER",
		"mail.smtp_password":            "SMTP_PASSWORD",
		"mail.from_address":             "SMTP_FROM",
		"mail.disable_security":         "MAIL_DISABLE_SECURITY",
		"mail.tls_insecure_skip_verify": "MAIL_TLS_INSECURE_SKIP_VERIFY",

		"auth_service.addr": "AUTH_SERVICE_ADDR",

		"grpc.server_addr": "GRPC_SERVER_ADDR",
		"grpc.event_addr":  "GRPC_EVENT_ADDR",

		"digest.max_records": "DIGEST_MAX_RECORDS",
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
		do.Provide(i, func(_ do.Injector) (*Config, error) {
			return initConfig(configPath)
		})
	}
}
