// Package testutil_test provides testing helpers for the assern project.
package testutil_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// TempDir creates a temporary directory and returns its path.
// Uses t.Cleanup() for automatic cleanup.
func TempDir(t *testing.T) string {
	t.Helper()

	return t.TempDir()
}

// CreateFile creates a file with content in a temp directory.
func CreateFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

// Logger returns a logger for testing.
func Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// CreateAssernDir creates a .assern directory in the given parent.
func CreateAssernDir(t *testing.T, parent string) string {
	t.Helper()
	path := filepath.Join(parent, ".assern")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}

	return path
}

// DiscardLogger returns a logger that discards all output.
func DiscardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
