package config

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	GRPC   GRPCConfig   `mapstructure:"grpc"`
	Log    LogConfig    `mapstructure:"log"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
}

type GRPCConfig struct {
	ServerAddr string `mapstructure:"server_addr"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}
