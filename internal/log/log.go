// Package log provides in-memory logging for UI display.
package log

import (
	"fmt"
	"sync"
)

// Level represents log levels.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Output interface for log messages.
type Output interface {
	Write(level, message string)
}

// Logger handles log messages.
type Logger struct {
	mu     sync.RWMutex
	level  Level
	output Output
}

var defaultLogger = &Logger{level: LevelInfo}

// Default returns the default logger.
func Default() *Logger {
	return defaultLogger
}

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput sets the output destination for log messages.
func (l *Logger) SetOutput(output Output) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = output
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, "DEBUG", format, args...)
}

// Info logs an info message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, "INFO", format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, "WARN", format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, "ERROR", format, args...)
}

func (l *Logger) log(level Level, levelStr, format string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if level < l.level {
		return
	}

	if l.output == nil {
		return
	}

	message := fmt.Sprintf(format, args...)
	l.output.Write(levelStr, message)
}

// Package-level functions use the default logger
func Debug(format string, args ...interface{}) { defaultLogger.Debug(format, args...) }
func Info(format string, args ...interface{})  { defaultLogger.Info(format, args...) }
func Warn(format string, args ...interface{})  { defaultLogger.Warn(format, args...) }
func Error(format string, args ...interface{}) { defaultLogger.Error(format, args...) }
