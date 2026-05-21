package temporal

import "go.uber.org/zap"

type TemporalLogger struct {
	logger *zap.SugaredLogger
}

func (l *TemporalLogger) Debug(msg string, keyvals ...any) {
	l.logger.Debugw(msg, keyvals...)
}

func (l *TemporalLogger) Info(msg string, keyvals ...any) {
	l.logger.Infow(msg, keyvals...)
}

func (l *TemporalLogger) Warn(msg string, keyvals ...any) {
	l.logger.Warnw(msg, keyvals...)
}

func (l *TemporalLogger) Error(msg string, keyvals ...any) {
	l.logger.Errorw(msg, keyvals...)
}
