package config

import (
	"fmt"

	"github.com/samber/do/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func newZapLogger(cfg *Config, isDev bool) (*zap.Logger, error) {

	if isDev {
		return zap.NewDevelopment()
	}

	level := cfg.Log.Level
	if level == "" {
		level = "info"
	}

	zapCfg := zap.NewProductionConfig()
	if zapLevel, err := zap.ParseAtomicLevel(level); err != nil {
		zapCfg.Level = zapLevel
	}

	lumberjackLogger := &lumberjack.Logger{
		Filename:   "uptime-monitor.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	options := []zap.Option{
		zap.Hooks(func(e zapcore.Entry) error {
			_, err := fmt.Fprintf(lumberjackLogger, "%+v", e)
			return err
		}),
	}

	return zapCfg.Build(options...)
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
