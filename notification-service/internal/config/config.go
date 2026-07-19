package config

type Config struct {
	DB            DBConfig     `mapstructure:"db"`
	Temporal      TemporalCfg  `mapstructure:"temporal"`
	Mail          MailConfig   `mapstructure:"mail"`
	Log           LogConfig    `mapstructure:"log"`
	Server        ServerConfig `mapstructure:"server"`
	AuthService   ServiceAddr  `mapstructure:"auth_service"`
	ServerService ServiceAddr  `mapstructure:"server_service"`
	OntimeService ServiceAddr  `mapstructure:"ontime_service"`
	GRPC          GRPCConfig   `mapstructure:"grpc"`
	Digest        DigestConfig `mapstructure:"digest"`
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

type TemporalCfg struct {
	Host            string `mapstructure:"host"`
	TaskQueue       string `mapstructure:"task_queue"`
	WorkflowName    string `mapstructure:"workflow_name"`
	DigestTaskQueue string `mapstructure:"digest_task_queue"`
}

type MailConfig struct {
	SMTPHost              string `mapstructure:"smtp_host"`
	SMTPPort              int    `mapstructure:"smtp_port"`
	SMTPUser              string `mapstructure:"smtp_user"`
	SMTPPassword          string `mapstructure:"smtp_password"`
	FromAddress           string `mapstructure:"from_address"`
	DisableSecurity       bool   `mapstructure:"disable_security"`
	TLSInsecureSkipVerify bool   `mapstructure:"tls_insecure_skip_verify"`
}

type ServiceAddr struct {
	Addr string `mapstructure:"addr"`
}

type GRPCConfig struct {
	ServerAddr string `mapstructure:"server_addr"`
	EventAddr  string `mapstructure:"event_addr"`
}

type DigestConfig struct {
	MaxRecords int `mapstructure:"max_records"`
}
