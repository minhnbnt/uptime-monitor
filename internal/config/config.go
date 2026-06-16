package config

type Config struct {
	DB       DBConfig       `mapstructure:"db"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Token    TokenCfg       `mapstructure:"token"`
	Argon2   Argon2Cfg      `mapstructure:"argon2"`
	Temporal TemporalConfig `mapstructure:"temporal"`
	Log      LogConfig      `mapstructure:"log"`
}

type TemporalConfig struct {
	Host         string `mapstructure:"host"`
	TaskQueue    string `mapstructure:"task_queue"`
	WorkflowName string `mapstructure:"workflow_name"`
}
