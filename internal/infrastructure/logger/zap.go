package logger

import (
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

type ZapLogger struct {
	logger *zap.Logger
}

func RegisterLogger(i do.Injector) {
	do.Provide(i, func(i do.Injector) (Logger, error) {
		return &ZapLogger{
			logger: do.MustInvoke[*zap.Logger](i),
		}, nil
	})
}

func toZapFields(fields []Field) []zap.Field {

	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, zap.Any(f.Key, f.Value))
	}

	return zapFields
}

func (l *ZapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, toZapFields(fields)...)
}

func (l *ZapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, toZapFields(fields)...)
}

func (l *ZapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, toZapFields(fields)...)
}

func (l *ZapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, toZapFields(fields)...)
}

func (l *ZapLogger) Fatal(msg string, fields ...Field) {
	l.logger.Fatal(msg, toZapFields(fields)...)
}

func (l *ZapLogger) With(fields ...Field) Logger {
	return &ZapLogger{logger: l.logger.With(toZapFields(fields)...)}
}

func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}
