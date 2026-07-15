package config

import (
	"flag"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"
)

const configPathKey = "config"

func RegisterConfigPath(configPath string) func(do.Injector) {
	return func(i do.Injector) {
		do.ProvideValue[viper.Viper](i, func() viper.Viper {
			v := viper.New()
			v.SetConfigFile(configPath)
			v.AddConfigPath(".")
			v.SetEnvPrefix("NOTIFICATION")
			v.AutomaticEnv()

			if err := v.ReadInConfig(); err != nil {
				panic(err)
			}

			return *v
		}())
	}
}

func RegisterConfigPathFlag() (string, error) {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	if *configPath == "" {
		return "", nil
	}

	return *configPath, nil
}

func RegisterConfig(cfg *Config) func(do.Injector) {
	return func(i do.Injector) {
		do.ProvideValue(i, cfg)

		initViperFrom(cfg)
	}
}

func initViperFrom(cfg *Config) {
	set := map[string]any{
		"server.port":         cfg.Server.Port,
		"db.host":             cfg.DB.Host,
		"db.port":             cfg.DB.Port,
		"db.user":             cfg.DB.User,
		"db.password":         cfg.DB.Password,
		"db.dbname":           cfg.DB.DBName,
		"temporal.host_port":  cfg.Temporal.HostPort,
		"mail.host":           cfg.Mail.Host,
		"mail.port":           cfg.Mail.Port,
		"mail.username":       cfg.Mail.Username,
		"mail.password":       cfg.Mail.Password,
		"mail.from_address":   cfg.Mail.FromAddress,
		"log.level":           cfg.Log.Level,
		"auth_service.addr":   cfg.AuthService.Addr,
		"server_service.addr": cfg.ServerService.Addr,
		"ontime_service.addr": cfg.OntimeService.Addr,
	}

	v := viper.New()
	for k, val := range set {
		v.SetDefault(k, val)
	}
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{Port: "8085"},
		DB: DBConfig{
			Host: "localhost", Port: "5432", User: "notification",
			Password: "notification", DBName: "notification",
		},
		Temporal: TemporalCfg{
			HostPort: "localhost:7233", DigestTaskQueue: "digest-task-queue",
		},
		Mail: MailConfig{
			Host: "localhost", Port: 1025, FromAddress: "noreply@uptime-monitor.local",
		},
		Log: LogConfig{Level: "info"},
		AuthService:   ServiceAddr{Addr: "http://auth-service:8081"},
		ServerService: ServiceAddr{Addr: "grpc://server:50051"},
		OntimeService: ServiceAddr{Addr: "http://ontime-service:8084"},
	}
}

func LoadConfig(configPath string) *Config {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.AddConfigPath(".")
	v.SetEnvPrefix("NOTIFICATION")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}

	cfg := defaultConfig()
	if err := v.Unmarshal(cfg); err != nil {
		panic(err)
	}

	return cfg
}
