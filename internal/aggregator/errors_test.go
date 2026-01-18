package aggregator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "ErrConfigRequired",
			err:  ErrConfigRequired,
			want: "config is required",
		},
		{
			name: "ErrNoServers",
			err:  ErrNoServers,
			want: "no MCP servers configured",
		},
		{
			name: "ErrServerNotStarted",
			err:  ErrServerNotStarted,
			want: "server not started",
		},
		{
			name: "ErrServerAlreadyStarted",
			err:  ErrServerAlreadyStarted,
			want: "server already started",
		},
		{
			name: "ErrServerNotFound",
			err:  ErrServerNotFound,
			want: "server not found",
		},
		{
			name: "ErrAllServersFailed",
			err:  ErrAllServersFailed,
			want: "all servers failed to start",
		},
		{
			name: "ErrInvalidTransport",
			err:  ErrInvalidTransport,
			want: "server must have either command (stdio) or url (http/sse)",
		},
		{
			name: "ErrOAuthRequired",
			err:  ErrOAuthRequired,
			want: "OAuth configuration required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.err.Error() != tt.want {
				t.Errorf("%s.Error() = %q, want %q", tt.name, tt.err.Error(), tt.want)
			}
		})
	}
}

func TestSentinelErrorsAreComparable(t *testing.T) {
	t.Parallel()

	// Test that sentinel errors can be detected with errors.Is()
	tests := []struct {
		name       string
		wrapped    error
		target     error
		shouldFind bool
	}{
		{
			name:       "wrapped ErrNoServers",
			wrapped:    fmt.Errorf("failed to start: %w", ErrNoServers),
			target:     ErrNoServers,
			shouldFind: true,
		},
		{
			name:       "wrapped ErrServerNotStarted",
			wrapped:    fmt.Errorf("call failed: %w", ErrServerNotStarted),
			target:     ErrServerNotStarted,
			shouldFind: true,
		},
		{
			name:       "double wrapped ErrAllServersFailed",
			wrapped:    fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", ErrAllServersFailed)),
			target:     ErrAllServersFailed,
			shouldFind: true,
		},
		{
			name:       "different errors don't match",
			wrapped:    fmt.Errorf("some error: %w", ErrNoServers),
			target:     ErrServerNotFound,
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			found := errors.Is(tt.wrapped, tt.target)
			if found != tt.shouldFind {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", tt.wrapped, tt.target, found, tt.shouldFind)
			}
		})
	}
}

func TestCommandNotFoundError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		err          *CommandNotFoundError
		wantInMsg    string
		notWantInMsg string
	}{
		{
			name: "absolute path missing",
			err: &CommandNotFoundError{
				ServerName: "test-server",
				Command:    "/nonexistent/path/to/npx",
				Type:       "absolute_path_missing",
			},
			wantInMsg:    "server test-server: command '/nonexistent/path/to/npx' not found",
			notWantInMsg: "Searched in PATH",
		},
		{
			name: "command not in PATH",
			err: &CommandNotFoundError{
				ServerName: "my-server",
				Command:    "npx",
				Type:       "command_not_in_path",
				Suggestion: "Install npx or use absolute path in config",
			},
			wantInMsg:    "Searched in PATH directories",
			notWantInMsg: "",
		},
		{
			name: "with suggestion",
			err: &CommandNotFoundError{
				ServerName: "sequential-thinking",
				Command:    "uvx",
				Type:       "command_not_in_path",
				Suggestion: "Install uvx or use absolute path in config",
			},
			wantInMsg:    "Suggestion: Install uvx or use absolute path in config",
			notWantInMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := tt.err.Error()
			if !strings.Contains(msg, tt.wantInMsg) {
				t.Errorf("Error message does not contain expected substring %q\nGot: %s", tt.wantInMsg, msg)
			}
			if tt.notWantInMsg != "" && strings.Contains(msg, tt.notWantInMsg) {
				t.Errorf("Error message contains unexpected substring %q\nGot: %s", tt.notWantInMsg, msg)
			}
		})
	}
}

func TestInitializationError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       *InitializationError
		wantInMsg string
	}{
		{
			name: "timeout error",
			err: &InitializationError{
				ServerName: "slow-server",
				Command:    "npx",
				Transport:  "stdio",
				Timeout:    60000000000, // 60 seconds
				IsTimeout:  true,
			},
			wantInMsg: "initialization timeout after 1m0s",
		},
		{
			name: "non-timeout error",
			err: &InitializationError{
				ServerName: "failing-server",
				Command:    "/path/to/server",
				Transport:  "stdio",
				Underlying: errors.New("connection refused"),
				IsTimeout:  false,
			},
			wantInMsg: "initialization failed",
		},
		{
			name: "timeout with all fields",
			err: &InitializationError{
				ServerName: "test-server",
				Command:    "npx",
				Transport:  "stdio",
				Timeout:    120000000000, // 2 minutes
				Underlying: context.DeadlineExceeded,
				IsTimeout:  true,
			},
			wantInMsg: "Possible causes:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := tt.err.Error()
			if !strings.Contains(msg, tt.wantInMsg) {
				t.Errorf("Error message does not contain expected substring %q\nGot: %s", tt.wantInMsg, msg)
			}

			// Verify timeout errors include helpful info
			if tt.err.IsTimeout {
				if !strings.Contains(msg, "Command:") {
					t.Error("Timeout error should include command information")
				}
				if !strings.Contains(msg, "Transport:") {
					t.Error("Timeout error should include transport information")
				}
				if !strings.Contains(msg, "Solutions:") {
					t.Error("Timeout error should include solutions")
				}
			}
		})
	}
}
