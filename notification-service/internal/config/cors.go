package config

import (
	"github.com/rs/cors"
	"github.com/samber/do/v2"
)

type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	MaxAge           int      `mapstructure:"max_age"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
}

func NewCORS(config *CORSConfig) *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins:   config.AllowedOrigins,
		AllowedMethods:   config.AllowedMethods,
		AllowedHeaders:   config.AllowedHeaders,
		MaxAge:           config.MaxAge,
		AllowCredentials: config.AllowCredentials,
	})
}

func RegisterCORS(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*cors.Cors, error) {
		config := do.MustInvoke[*Config](i)
		return NewCORS(&config.CORS), nil
	})
}
