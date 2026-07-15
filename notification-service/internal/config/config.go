package config

type Config struct {
	DB      DBConfig      `mapstructure:"db"`
	Temporal TemporalCfg  `mapstructure:"temporal"`
	Mail    MailConfig    `mapstructure:"mail"`
	Log     LogConfig     `mapstructure:"log"`
	Server  ServerConfig  `mapstructure:"server"`

	AuthService  ServiceAddr `mapstructure:"auth_service"`
	ServerService ServiceAddr `mapstructure:"server_service"`
	OntimeService ServiceAddr `mapstructure:"ontime_service"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
}

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type MailConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	FromAddress  string `mapstructure:"from_address"`
}

type TemporalCfg struct {
	HostPort        string `mapstructure:"host_port"`
	DigestTaskQueue string `mapstructure:"digest_task_queue"`
}

type ServiceAddr struct {
	Addr string `mapstructure:"addr"`
}
