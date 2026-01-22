package transport

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// StdioRedirector captures stdout to prevent library output from interfering
// with MCP protocol communication.
type StdioRedirector struct {
	originalStdout *os.File
	writer         io.Writer
	logger         *slog.Logger
}

// NewStdioRedirector creates a new stdio redirector.
func NewStdioRedirector(logger *slog.Logger) *StdioRedirector {
	return &StdioRedirector{
		originalStdout: os.Stdout,
		logger:         logger,
	}
}

// Redirect redirects stdout to the given writer (typically os.Stderr or a log file).
func (r *StdioRedirector) Redirect(w io.Writer) error {
	r.writer = w

	// Create a pipe
	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}

	// Replace stdout with the write end of the pipe
	os.Stdout = pw

	// Copy pipe output to the writer in a goroutine
	go func() {
		_, err := io.Copy(w, pr)
		if err != nil {
			r.logger.Debug("stdout redirect copy error", "error", err)
		}
	}()

	return nil
}

// Restore restores the original stdout.
func (r *StdioRedirector) Restore() {
	if r.originalStdout != nil {
		os.Stdout = r.originalStdout
	}
}

// DiscardLogger returns a logger that discards all output.
func DiscardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// StderrLogger returns a logger that writes to stderr.
func StderrLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}

// FileLogger returns a logger that writes to a file.
func FileLogger(path string, level slog.Level) (*slog.Logger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	return slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: level,
	})), nil
}

// ParseLogLevel converts a string log level to slog.Level.
// It is case-insensitive and trims whitespace.
func ParseLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug", "trace":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
