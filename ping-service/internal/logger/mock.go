package logger

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"sync"
)

func NewMockLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type captureHandler struct {
	mu      sync.RWMutex
	records []slog.Record
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h *captureHandler) WithGroup(string) slog.Handler {
	return h
}

func (h *captureHandler) Has(level slog.Level) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return slices.ContainsFunc(h.records, func(record slog.Record) bool {
		return record.Level == level
	})
}

func (h *captureHandler) HasWarn() bool {
	return h.Has(slog.LevelWarn)
}

func (h *captureHandler) HasError() bool {
	return h.Has(slog.LevelError)
}

func NewCapturingLogger() (*slog.Logger, *captureHandler) {
	h := &captureHandler{}
	return slog.New(h), h
}
