package config

type Config struct {
	DB     DBConfig    `mapstructure:"db"`
	Redis  RedisConfig `mapstructure:"redis"`
	Server ServerCfg   `mapstructure:"server"`
	Log    LogConfig   `mapstructure:"log"`
	GRPC   GRPCConfig  `mapstructure:"grpc"`
}

type GRPCConfig struct {
	ServerAddr string `mapstructure:"server_addr"`
}

type ServerCfg struct {
	Port string `mapstructure:"port"`
}
