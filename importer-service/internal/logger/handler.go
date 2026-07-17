package logger

import (
	"context"
	"log/slog"
	"runtime/debug"
)

type stackHandler struct {
	slog.Handler
}

func (h stackHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		r.AddAttrs(slog.String("stack", string(debug.Stack())))
	}
	return h.Handler.Handle(ctx, r)
}

func WithStack(h slog.Handler) slog.Handler {
	return stackHandler{h}
}
