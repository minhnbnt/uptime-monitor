package config

import (
	"log/slog"
	"os"

	"github.com/samber/do/v2"
)

func RegisterLogger(dev bool) func(do.Injector) {
	return func(i do.Injector) {
		opts := &slog.HandlerOptions{Level: slog.LevelInfo}
		var handler slog.Handler
		if dev {
			handler = slog.NewTextHandler(os.Stdout, opts)
		} else {
			handler = slog.NewJSONHandler(os.Stdout, opts)
		}

		do.ProvideValue(i, slog.New(handler))
	}
}
