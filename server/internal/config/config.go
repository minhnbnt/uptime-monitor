package config

type Config struct {
	DB        DBConfig       `mapstructure:"db"`
	Redis     RedisConfig    `mapstructure:"redis"`
	JWT       JWTConfig      `mapstructure:"jwt"`
	Token     TokenCfg       `mapstructure:"token"`
	Argon2    Argon2Cfg      `mapstructure:"argon2"`
	Temporal  TemporalConfig `mapstructure:"temporal"`
	Scheduler SchedulerCfg   `mapstructure:"scheduler"`
	Log       LogConfig      `mapstructure:"log"`
	Mail      MailConfig     `mapstructure:"mail"`
	GRPC      GRPCConfig     `mapstructure:"grpc"`
}

type GRPCConfig struct {
	Port string `mapstructure:"port"`
}

type SchedulerCfg struct {
	Backend string `mapstructure:"backend"`
}

type TemporalConfig struct {
	Host            string `mapstructure:"host"`
	TaskQueue       string `mapstructure:"task_queue"`
	WorkflowName    string `mapstructure:"workflow_name"`
	DigestTaskQueue string `mapstructure:"digest_task_queue"`
}
