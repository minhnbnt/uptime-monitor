package config

type Config struct {
	Redis      RedisConfig `mapstructure:"redis"`
	Server     ServerCfg   `mapstructure:"server"`
	Log        LogConfig   `mapstructure:"log"`
	GRPC       GRPCConfig  `mapstructure:"grpc"`
	Kubeconfig string      `mapstructure:"kubeconfig"`
	K8s        K8sConfig   `mapstructure:"k8s"`
}

type K8sConfig struct {
	QPS   float32 `mapstructure:"qps"`
	Burst int     `mapstructure:"burst"`
}

type GRPCConfig struct {
	Port       string `mapstructure:"port"`
	ServerAddr string `mapstructure:"server_addr"`
	EventAddr  string `mapstructure:"event_addr"`
}

type ServerCfg struct {
	Port string `mapstructure:"port"`
}
