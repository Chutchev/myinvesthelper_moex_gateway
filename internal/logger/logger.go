package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog.Logger with consistent configuration
type Logger struct {
	*slog.Logger
	errorLog *slog.Logger
}

// New creates a new structured logger with the specified level
func New(level string) *Logger {
	lvl := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}

	// Main logger (stdout)
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: false,
	})

	// Error logger (file)
	errorFile, err := os.OpenFile("error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Fallback to stdout if file can't be opened
		errorFile = os.Stderr
	}

	errorHandler := slog.NewTextHandler(errorFile, &slog.HandlerOptions{
		Level:     slog.LevelError,
		AddSource: true, // Include source file and line in error logs
	})

	return &Logger{
		Logger:   slog.New(handler),
		errorLog: slog.New(errorHandler),
	}
}

// NewJSON creates a new structured logger with JSON output
func NewJSON(level string) *Logger {
	lvl := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}

	// Main logger (stdout)
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: false,
	})

	// Error logger (file)
	errorFile, err := os.OpenFile("error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		errorFile = os.Stderr
	}

	errorHandler := slog.NewJSONHandler(errorFile, &slog.HandlerOptions{
		Level:     slog.LevelError,
		AddSource: true, // Include source file and line in error logs
	})

	return &Logger{
		Logger:   slog.New(handler),
		errorLog: slog.New(errorHandler),
	}
}

// Error logs at error level to both stdout and error file
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
	l.errorLog.Error(msg, args...)
}

// Info logs at info level to stdout only
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Debug logs at debug level to stdout only
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Warn logs at warn level to stdout only
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}
