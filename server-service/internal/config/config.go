package config

type Config struct {
	DB    DBConfig    `mapstructure:"db"`
	Redis RedisConfig `mapstructure:"redis"`
	Log   LogConfig   `mapstructure:"log"`
	GRPC  GRPCConfig  `mapstructure:"grpc"`
}

type GRPCConfig struct {
	Port       string `mapstructure:"port"`
	ServerAddr string `mapstructure:"server_addr"`
	EventAddr  string `mapstructure:"event_addr"`
}
