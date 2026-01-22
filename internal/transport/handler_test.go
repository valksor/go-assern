package transport_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valksor/go-assern/internal/transport"
)

func TestDiscardLogger(t *testing.T) {
	t.Parallel()

	logger := transport.DiscardLogger()

	if logger == nil {
		t.Fatal("DiscardLogger() returned nil")
	}

	// Should not panic when logging
	logger.Info("test message")
	logger.Debug("debug message")
	logger.Warn("warning message")
	logger.Error("error message")
}

func TestStderrLogger(t *testing.T) {
	t.Parallel()

	logger := transport.StderrLogger(slog.LevelInfo)

	if logger == nil {
		t.Fatal("StderrLogger() returned nil")
	}

	// Should not panic when logging
	logger.Info("test message")
	logger.Debug("debug message")
	logger.Warn("warning message")
	logger.Error("error message")
}

func TestFileLogger(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := transport.FileLogger(logPath, slog.LevelInfo)
	if err != nil {
		t.Fatalf("FileLogger() error = %v", err)
	}

	if logger == nil {
		t.Fatal("FileLogger() returned nil")
	}

	// Log some messages
	testMsg := "test log message"
	logger.Info(testMsg)

	// Flush/handshake - the file should be written when logger is released

	// Read the log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), testMsg) {
		t.Errorf("Log file does not contain test message. Got: %s", string(content))
	}
}

func TestFileLogger_CreatePath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// Create parent directory first since FileLogger doesn't do it automatically
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(subDir, "test.log")

	_, err := transport.FileLogger(logPath, slog.LevelInfo)
	if err != nil {
		t.Fatalf("FileLogger() error = %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(logPath); err != nil {
		t.Errorf("Log file was not created: %v", err)
	}
}

func TestParseLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"trace", "trace", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"warning", "warning", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"unknown", "unknown", slog.LevelInfo},
		{"empty", "", slog.LevelInfo},
		// Case insensitivity tests
		{"DEBUG uppercase", "DEBUG", slog.LevelDebug},
		{"INFO uppercase", "INFO", slog.LevelInfo},
		{"WARN uppercase", "WARN", slog.LevelWarn},
		{"ERROR uppercase", "ERROR", slog.LevelError},
		{"Debug mixed", "Debug", slog.LevelDebug},
		{"Warning mixed", "Warning", slog.LevelWarn},
		// Whitespace handling
		{"debug with spaces", "  debug  ", slog.LevelDebug},
		{"info with tabs", "\tinfo\t", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := transport.ParseLogLevel(tt.input)
			if got != tt.want {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewStdioRedirector(t *testing.T) {
	logger := transport.DiscardLogger()
	redirector := transport.NewStdioRedirector(logger)

	if redirector == nil {
		t.Fatal("NewStdioRedirector() returned nil")
	}
}

func TestStdioRedirector_Redirect(t *testing.T) {
	// Save original stdout
	originalStdout := os.Stdout

	// Create a buffer to capture redirected output
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, nil))
	redirector := transport.NewStdioRedirector(logger)

	// Redirect stdout to our buffer
	if err := redirector.Redirect(&buf); err != nil {
		t.Fatalf("Redirect() error = %v", err)
	}

	// Restore stdout after test
	defer func() {
		redirector.Restore()
		os.Stdout = originalStdout
	}()

	// Write to stdout (which is now redirected)
	// Note: This is a basic test - in practice, the goroutine copy
	// might not complete immediately
}

func TestStdioRedirector_Restore(t *testing.T) {
	originalStdout := os.Stdout
	logger := transport.DiscardLogger()
	redirector := transport.NewStdioRedirector(logger)

	// Redirect
	var buf bytes.Buffer
	if err := redirector.Redirect(&buf); err != nil {
		t.Fatalf("Redirect() error = %v", err)
	}

	// Restore
	redirector.Restore()

	// Verify stdout is restored
	if os.Stdout != originalStdout {
		t.Error("Restore() did not restore original stdout")
	}

	// Calling Restore again should be safe (no panic)
	redirector.Restore()
}

func TestStderrLogger_Levels(t *testing.T) {
	t.Parallel()

	// Test that different log levels work
	levels := []slog.Level{
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			t.Parallel()

			logger := transport.StderrLogger(level)
			if logger == nil {
				t.Error("StderrLogger() returned nil")
			}
		})
	}
}
