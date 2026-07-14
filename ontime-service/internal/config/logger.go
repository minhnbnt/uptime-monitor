package config

import (
	"io"
	"log/slog"
	"os"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/logger"
)

func newLogger(cfg *Config, isDev bool) *slog.Logger {

	level := cfg.Log.Level
	if level == "" {
		level = "info"
	}

	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     lvl,
		AddSource: true,
	}

	var handler slog.Handler
	if isDev {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(io.MultiWriter(os.Stdout), opts)
	}

	return slog.New(logger.WithStack(handler))
}

func RegisterLogger(isDev bool) func(do.Injector) {
	return func(i do.Injector) {
		do.Provide(i, func(i do.Injector) (*slog.Logger, error) {
			cfg := do.MustInvoke[*Config](i)
			return newLogger(cfg, isDev), nil
		})
	}
}
