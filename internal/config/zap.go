package config

import (
	"os"

	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

func newZapLogger(i do.Injector) (*zap.Logger, error) {

	if os.Getenv("APP_ENV") == "development" {
		return zap.NewDevelopment()
	}

	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "info"
	}

	zapLevel, err := zap.ParseAtomicLevel(level)
	if err != nil {
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	cfg := zap.Config{
		Level:            zapLevel,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return cfg.Build()
}

func RegisterZapLogger(i do.Injector) {
	do.Provide(i, newZapLogger)
}
