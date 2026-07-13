package logger

import (
	"io"
	"log/slog"
)

func NewMockLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
