// Package log provides a no-op logger.
package log

// Level represents log levels.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger is a no-op logger.
type Logger struct{}

// Default returns the default logger.
func Default() *Logger {
	return &Logger{}
}

// SetLevel sets the log level (no-op).
func (l *Logger) SetLevel(level Level) {}

// Debug logs a debug message (no-op).
func (l *Logger) Debug(format string, args ...interface{}) {}

// Info logs an info message (no-op).
func (l *Logger) Info(format string, args ...interface{}) {}

// Warn logs a warning message (no-op).
func (l *Logger) Warn(format string, args ...interface{}) {}

// Error logs an error message (no-op).
func (l *Logger) Error(format string, args ...interface{}) {}

// Package-level functions
func Debug(format string, args ...interface{}) {}
func Info(format string, args ...interface{})  {}
func Warn(format string, args ...interface{})  {}
func Error(format string, args ...interface{}) {}
