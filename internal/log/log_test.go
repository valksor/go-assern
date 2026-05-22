package log

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestConfigure(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantNil bool
	}{
		{
			name:    "default options",
			opts:    Options{},
			wantNil: false,
		},
		{
			name: "with output",
			opts: Options{
				Output: &bytes.Buffer{},
			},
			wantNil: false,
		},
		{
			name: "with JSON",
			opts: Options{
				Output: &bytes.Buffer{},
				JSON:   true,
			},
			wantNil: false,
		},
		{
			name: "with verbose",
			opts: Options{
				Verbose: true,
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Configure(tt.opts)
			logger := Logger()
			if (logger == nil) != tt.wantNil {
				t.Errorf("Configure() logger = %v, wantNil %v", logger, tt.wantNil)
			}
		})
	}
}

func TestSetLevel(t *testing.T) {
	levels := []Level{LevelDebug, LevelInfo, LevelWarn, LevelError}
	for _, level := range levels {
		SetLevel(level)
		// Just ensure it doesn't panic
	}
}

func TestEnableDebug(t *testing.T) {
	EnableDebug()
	// Just ensure it doesn't panic
}

func TestWith(t *testing.T) {
	logger := With("key", "value")
	if logger == nil {
		t.Error("With() returned nil logger")
	}
}

func TestLoggingFunctions(t *testing.T) {
	tests := []struct {
		name  string
		fn    func(msg string, args ...any)
		level Level
	}{
		{"Debug", Debug, LevelDebug},
		{"Info", Info, LevelInfo},
		{"Warn", Warn, LevelWarn},
		{"Error", Error, LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Configure(Options{Output: &buf, Level: tt.level})
			tt.fn("test message", "key", "value")

			if buf.Len() == 0 {
				t.Error("Logging function produced no output")
			}
		})
	}
}

func TestLoggingFunctionsContext(t *testing.T) {
	tests := []struct {
		name  string
		fn    func(ctx context.Context, msg string, args ...any)
		level Level
	}{
		{"DebugContext", DebugContext, LevelDebug},
		{"InfoContext", InfoContext, LevelInfo},
		{"WarnContext", WarnContext, LevelWarn},
		{"ErrorContext", ErrorContext, LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Configure(Options{Output: &buf, Level: tt.level})
			ctx := context.Background()
			tt.fn(ctx, "test message", "key", "value")

			if buf.Len() == 0 {
				t.Error("Logging function produced no output")
			}
		})
	}
}

func TestErr(t *testing.T) {
	err := errors.New("test error")
	attr := Err(err)

	if attr.Key != "error" {
		t.Errorf("Err() key = %v, want %v", attr.Key, "error")
	}

	if !strings.Contains(attr.Value.String(), "test error") {
		t.Errorf("Err() value = %v, want to contain 'test error'", attr.Value)
	}
}

func TestLevelTypes(t *testing.T) {
	// Just ensure the type aliases work correctly
	level := LevelInfo
	if level != slog.LevelInfo {
		t.Errorf("Level type alias not working correctly")
	}
}

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	Configure(Options{
		Output: &buf,
		JSON:   true,
	})

	Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, `"test message"`) {
		t.Errorf("JSON output doesn't contain expected message, got: %s", output)
	}
	if !strings.Contains(output, `"key"`) {
		t.Errorf("JSON output doesn't contain expected key, got: %s", output)
	}
}

func TestTextOutput(t *testing.T) {
	var buf bytes.Buffer
	Configure(Options{
		Output: &buf,
		JSON:   false,
	})

	Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Text output doesn't contain expected message, got: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("Text output doesn't contain expected key, got: %s", output)
	}
}

func TestVerboseEnablesDebug(t *testing.T) {
	var buf bytes.Buffer
	Configure(Options{
		Output:  &buf,
		Verbose: true,
	})

	Debug("debug message")

	if buf.Len() == 0 {
		t.Error("Verbose option didn't enable debug logging")
	}
}
