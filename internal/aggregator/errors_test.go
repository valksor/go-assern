package aggregator

import (
	"errors"
	"fmt"
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
