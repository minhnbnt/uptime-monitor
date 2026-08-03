package config

type Config struct {
	Temporal TemporalCfg  `mapstructure:"temporal"`
	Mail     MailConfig   `mapstructure:"mail"`
	Log      LogConfig    `mapstructure:"log"`
	Server   ServerConfig `mapstructure:"server"`
	Auth     AuthConfig   `mapstructure:"auth"`
	GRPC     GRPCConfig   `mapstructure:"grpc"`
	Digest   DigestConfig `mapstructure:"digest"`
	CORS     CORSConfig   `mapstructure:"cors"`
}

type AuthConfig struct {
	Issuer       string `mapstructure:"issuer"`
	ServiceToken string `mapstructure:"service_token"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type TemporalCfg struct {
	Host            string `mapstructure:"host"`
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

type GRPCConfig struct {
	ServerAddr string `mapstructure:"server_addr"`
	EventAddr  string `mapstructure:"event_addr"`
}

type DigestConfig struct {
	MaxRecords int `mapstructure:"max_records"`
}
