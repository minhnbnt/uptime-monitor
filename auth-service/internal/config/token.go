package config

import (
	"time"

	"github.com/samber/do/v2"
)

type TokenConfig struct {
	accessTokenTTL     time.Duration
	refreshTokenTTL    time.Duration
	accessTokenIssuer  string
	refreshTokenIssuer string
}

func (c *TokenConfig) GetAccessTokenTTL() time.Duration {
	return c.accessTokenTTL
}

func (c *TokenConfig) GetRefreshTokenTTL() time.Duration {
	return c.refreshTokenTTL
}

func (c *TokenConfig) GetAccessTokenIssuer() string {
	return c.accessTokenIssuer
}

func (c *TokenConfig) GetRefreshTokenIssuer() string {
	return c.refreshTokenIssuer
}

func newTokenConfig(cfg *Config) (*TokenConfig, error) {
	accessTTL, err := time.ParseDuration(cfg.Token.AccessTTL)
	if err != nil {
		accessTTL = 15 * time.Minute
	}

	refreshTTL, err := time.ParseDuration(cfg.Token.RefreshTTL)
	if err != nil {
		refreshTTL = 7 * 24 * time.Hour
	}

	return &TokenConfig{
		accessTokenTTL:     accessTTL,
		refreshTokenTTL:    refreshTTL,
		accessTokenIssuer:  cfg.Token.AccessIssuer,
		refreshTokenIssuer: cfg.Token.RefreshIssuer,
	}, nil
}

func RegisterTokenConfig(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TokenConfig, error) {
		cfg := do.MustInvoke[*Config](i)
		return newTokenConfig(cfg)
	})
}
