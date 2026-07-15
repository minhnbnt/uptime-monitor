package config

type TokenCfg struct {
	AccessTTL     string `mapstructure:"access_ttl"`
	RefreshTTL    string `mapstructure:"refresh_ttl"`
	AccessIssuer  string `mapstructure:"access_issuer"`
	RefreshIssuer string `mapstructure:"refresh_issuer"`
}
