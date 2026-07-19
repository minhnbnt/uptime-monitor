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

type CaptureHandler struct {
	mu      sync.RWMutex
	records []slog.Record
}

func (h *CaptureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *CaptureHandler) Handle(_ context.Context, r slog.Record) error {

	h.mu.Lock()
	defer h.mu.Unlock()

	h.records = append(h.records, r.Clone())

	return nil
}

func (h *CaptureHandler) Has(level slog.Level) bool {

	h.mu.RLock()
	defer h.mu.RUnlock()

	return slices.ContainsFunc(h.records, func(record slog.Record) bool {
		return record.Level == level
	})
}

func (h *CaptureHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h *CaptureHandler) WithGroup(string) slog.Handler {
	return h
}

func (h *CaptureHandler) HasWarn() bool {
	return h.Has(slog.LevelWarn)
}

func (h *CaptureHandler) HasError() bool {
	return h.Has(slog.LevelError)
}

func NewCapturingLogger() (*slog.Logger, *CaptureHandler) {
	h := &CaptureHandler{}
	return slog.New(h), h
}
