package aggregator

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors for aggregator operations.
// These enable callers to check for specific error conditions using errors.Is().
var (
	// ErrConfigRequired indicates the config parameter is nil.
	ErrConfigRequired = errors.New("config is required")

	// ErrNoServers indicates no MCP servers are configured.
	ErrNoServers = errors.New("no MCP servers configured")

	// ErrServerNotStarted indicates an operation was attempted on a stopped server.
	ErrServerNotStarted = errors.New("server not started")

	// ErrServerAlreadyStarted indicates Start was called on a running server.
	ErrServerAlreadyStarted = errors.New("server already started")

	// ErrServerNotFound indicates a requested server does not exist.
	ErrServerNotFound = errors.New("server not found")

	// ErrAllServersFailed indicates every configured server failed to start.
	ErrAllServersFailed = errors.New("all servers failed to start")

	// ErrInvalidTransport indicates the server has no valid transport configuration.
	ErrInvalidTransport = errors.New("server must have either command (stdio) or url (http/sse)")

	// ErrOAuthRequired indicates OAuth configuration is missing for an OAuth transport.
	ErrOAuthRequired = errors.New("OAuth configuration required")
)

// CommandNotFoundError is returned when a configured command cannot be found.
type CommandNotFoundError struct {
	ServerName string
	Command    string
	Err        error
	Type       string
	Suggestion string
}

func (e *CommandNotFoundError) Error() string {
	msg := fmt.Sprintf("server %s: command '%s' not found", e.ServerName, e.Command)
	if e.Suggestion != "" {
		msg += "\n  Suggestion: " + e.Suggestion
	}
	if e.Type == "command_not_in_path" {
		msg += "\n  Searched in PATH directories"
		msg += "\n  Fix: Use absolute path in config or ensure command is installed"
	}

	return msg
}

func (e *CommandNotFoundError) Unwrap() error { return e.Err }

// InitializationError is returned when server initialization fails.
type InitializationError struct {
	ServerName string
	Command    string
	Transport  string
	Timeout    time.Duration
	Underlying error
	IsTimeout  bool
}

func (e *InitializationError) Error() string {
	var msg string
	if e.IsTimeout {
		msg = fmt.Sprintf("server %s: initialization timeout after %v", e.ServerName, e.Timeout)
		msg += "\n  Command: " + e.Command
		msg += "\n  Transport: " + e.Transport
		msg += "\n\nPossible causes:"
		msg += "\n  1. Command is slow to start (first-time package download?)"
		msg += "\n  2. Network connectivity issues"
		msg += "\n  3. Resource constraints (CPU/memory)"
		msg += "\n\nSolutions:"
		msg += "\n  - Increase timeout in config.yaml"
		msg += "\n  - Pre-download packages manually"
		msg += "\n  - Check system resources"
	} else {
		msg = fmt.Sprintf("server %s: initialization failed", e.ServerName)
		msg += "\n  Command: " + e.Command
		msg += "\n  Transport: " + e.Transport
		msg += fmt.Sprintf("\n  Error: %v", e.Underlying)
	}

	return msg
}

func (e *InitializationError) Unwrap() error { return e.Underlying }
