package config

type Config struct {
	DB        DBConfig        `mapstructure:"db"`
	Redis     RedisConfig     `mapstructure:"redis"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Token     TokenCfg        `mapstructure:"token"`
	Argon2    Argon2Cfg       `mapstructure:"argon2"`
	Log       LogConfig       `mapstructure:"log"`
	Server    ServerConfig    `mapstructure:"server"`
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

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type JWTConfig struct {
	Key string `mapstructure:"key"`
}

type TokenCfg struct {
	AccessTTL     string `mapstructure:"access_ttl"`
	RefreshTTL    string `mapstructure:"refresh_ttl"`
	AccessIssuer  string `mapstructure:"access_issuer"`
	RefreshIssuer string `mapstructure:"refresh_issuer"`
}

type Argon2Cfg struct {
	Memory      uint32 `mapstructure:"memory"`
	Iterations  uint32 `mapstructure:"iterations"`
	Parallelism uint8  `mapstructure:"parallelism"`
	SaltLength  uint32 `mapstructure:"salt_length"`
	KeyLength   uint32 `mapstructure:"key_length"`
}
