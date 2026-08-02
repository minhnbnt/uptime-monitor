package config

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	GRPC   GRPCConfig   `mapstructure:"grpc"`
	Auth   AuthConfig   `mapstructure:"auth"`
	Log    LogConfig    `mapstructure:"log"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
}

type AuthConfig struct {
	Issuer string `mapstructure:"issuer"`
}

type GRPCConfig struct {
	ServerAddr string `mapstructure:"server_addr"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}
