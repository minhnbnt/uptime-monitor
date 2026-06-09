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

func newTokenConfig(i do.Injector) (*TokenConfig, error) {
	return &TokenConfig{
		accessTokenTTL:     15 * time.Minute,
		refreshTokenTTL:    7 * 24 * time.Hour,
		accessTokenIssuer:  "uptime-monitor",
		refreshTokenIssuer: "uptime-monitor-refresh",
	}, nil
}

func RegisterTokenConfig(i do.Injector) {
	do.Provide(i, newTokenConfig)
}
