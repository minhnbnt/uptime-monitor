package logger

type MockLogger struct {
	InfoCalled  bool
	WarnCalled  bool
	ErrorCalled bool
	DebugCalled bool
	FatalCalled bool
	LastMsg     string

	InfoFunc  func(msg string, fields ...Field)
	WarnFunc  func(msg string, fields ...Field)
	ErrorFunc func(msg string, fields ...Field)
	DebugFunc func(msg string, fields ...Field)
	FatalFunc func(msg string, fields ...Field)
	WithFunc  func(fields ...Field) Logger
}

func NewMockLogger() *MockLogger {
	m := &MockLogger{}
	m.InfoFunc = func(msg string, fields ...Field) {
		m.InfoCalled = true
		m.LastMsg = msg
	}
	m.WarnFunc = func(msg string, fields ...Field) {
		m.WarnCalled = true
		m.LastMsg = msg
	}
	m.ErrorFunc = func(msg string, fields ...Field) {
		m.ErrorCalled = true
		m.LastMsg = msg
	}
	m.DebugFunc = func(msg string, fields ...Field) {
		m.DebugCalled = true
		m.LastMsg = msg
	}
	m.FatalFunc = func(msg string, fields ...Field) {
		m.FatalCalled = true
		m.LastMsg = msg
	}
	m.WithFunc = func(fields ...Field) Logger {
		return m
	}
	return m
}

func (m *MockLogger) Info(msg string, fields ...Field) {
	if m.InfoFunc != nil {
		m.InfoFunc(msg, fields...)
	}
}

func (m *MockLogger) Warn(msg string, fields ...Field) {
	if m.WarnFunc != nil {
		m.WarnFunc(msg, fields...)
	}
}

func (m *MockLogger) Error(msg string, fields ...Field) {
	if m.ErrorFunc != nil {
		m.ErrorFunc(msg, fields...)
	}
}

func (m *MockLogger) Debug(msg string, fields ...Field) {
	if m.DebugFunc != nil {
		m.DebugFunc(msg, fields...)
	}
}

func (m *MockLogger) Fatal(msg string, fields ...Field) {
	if m.FatalFunc != nil {
		m.FatalFunc(msg, fields...)
	}
}

func (m *MockLogger) With(fields ...Field) Logger {
	if m.WithFunc != nil {
		return m.WithFunc(fields...)
	}
	return m
}
