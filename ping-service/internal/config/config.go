package config

type Config struct {
	DB        DBConfig       `mapstructure:"db"`
	Redis     RedisConfig    `mapstructure:"redis"`
	Server    ServerCfg      `mapstructure:"server"`
	Log       LogConfig      `mapstructure:"log"`
	Scheduler SchedulerCfg   `mapstructure:"scheduler"`
	GRPC      GRPCConfig     `mapstructure:"grpc"`
}

type GRPCConfig struct {
	ServerAddr string `mapstructure:"server_addr"`
}

type ServerCfg struct {
	Port string `mapstructure:"port"`
}

type SchedulerCfg struct {
	Backend string `mapstructure:"backend"`
}
