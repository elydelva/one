package fake

import "elydelva/one/internal/ports"

// Logger is a no-op logger that records calls for inspection.
type Logger struct {
	Entries []LogEntry
}

// LogEntry records a single log call.
type LogEntry struct {
	Level string
	Msg   string
	Attrs []any
}

// NewLogger creates a recording logger.
func NewLogger() *Logger { return &Logger{} }

func (l *Logger) record(level, msg string, attrs ...any) {
	l.Entries = append(l.Entries, LogEntry{Level: level, Msg: msg, Attrs: attrs})
}

func (l *Logger) Debug(msg string, attrs ...any) { l.record("debug", msg, attrs...) }
func (l *Logger) Info(msg string, attrs ...any)  { l.record("info", msg, attrs...) }
func (l *Logger) Warn(msg string, attrs ...any)  { l.record("warn", msg, attrs...) }
func (l *Logger) Error(msg string, attrs ...any) { l.record("error", msg, attrs...) }
func (l *Logger) With(_ ...any) ports.Logger     { return l }

var _ ports.Logger = (*Logger)(nil)
