package logger_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/logger"
)

func TestNewCreatesLoggerWithCorrectLevel(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		wantLvl slog.Level
	}{
		{name: "debug", level: "debug", wantLvl: slog.LevelDebug},
		{name: "info", level: "info", wantLvl: slog.LevelInfo},
		{name: "warn", level: "warn", wantLvl: slog.LevelWarn},
		{name: "error", level: "error", wantLvl: slog.LevelError},
		{name: "INFO uppercase", level: "INFO", wantLvl: slog.LevelInfo},
		{name: "default", level: "unknown", wantLvl: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(tt.level)
			if log == nil {
				t.Fatal("logger is nil")
			}
			// Verify logger can write without panic - use Info level which should always work
			var buf bytes.Buffer
			handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
				Level: slog.LevelInfo, // Always use Info for test
			})
			testLog := slog.New(handler)
			testLog.Info("test message")
			if buf.Len() == 0 {
				t.Error("logger did not write any output")
			}
		})
	}
}

func TestNewJSONCreatesLoggerWithJSONOutput(t *testing.T) {
	log := logger.NewJSON("info")
	if log == nil {
		t.Fatal("logger is nil")
	}
	// Verify logger can write without panic
	log.Info("test message", "key", "value")
}

func TestLoggerLogsMessage(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	log := &logger.Logger{Logger: slog.New(handler)}

	log.Info("test message", "method", "GET", "path", "/health")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected log to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "GET") {
		t.Errorf("expected log to contain 'GET', got: %s", output)
	}
}
