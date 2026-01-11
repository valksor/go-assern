package aggregator

import "errors"

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
