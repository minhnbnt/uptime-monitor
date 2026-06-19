package config

import (
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

func newZapLogger(cfg *Config, isDev bool) (*zap.Logger, error) {

	if isDev {
		return zap.NewDevelopment()
	}

	level := cfg.Log.Level
	if level == "" {
		level = "info"
	}

	zapLevel, err := zap.ParseAtomicLevel(level)
	if err != nil {
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	zapCfg := zap.Config{
		Level:            zapLevel,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return zapCfg.Build()
}

func RegisterZapLogger(isDev bool) func(do.Injector) {
	return func(i do.Injector) {
		do.Provide(i, func(i do.Injector) (*zap.Logger, error) {
			cfg := do.MustInvoke[*Config](i)
			return newZapLogger(cfg, isDev)
		})
	}
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}
