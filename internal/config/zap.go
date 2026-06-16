package config

import (
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

func newZapLogger(cfg *Config) (*zap.Logger, error) {

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

func RegisterZapLogger(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*zap.Logger, error) {
		cfg := do.MustInvoke[*Config](i)
		return newZapLogger(cfg)
	})
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}
