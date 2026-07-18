package config

type Config struct {
	Redis  RedisConfig  `mapstructure:"redis"`
	Server ServerCfg   `mapstructure:"server"`
	Log    LogConfig   `mapstructure:"log"`
	GRPC   GRPCConfig  `mapstructure:"grpc"`
}

type GRPCConfig struct {
	Port       string `mapstructure:"port"`
	ServerAddr string `mapstructure:"server_addr"`
	EventAddr  string `mapstructure:"event_addr"`
}

type ServerCfg struct {
	Port string `mapstructure:"port"`
}
