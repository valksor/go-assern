package aggregator

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/valksor/go-assern/internal/config"
)

// Server defines the interface for an MCP server that can be managed by the aggregator.
// This interface enables testing with mock implementations and supports
// future server types without modifying the aggregator.
type Server interface {
	// Name returns the server identifier.
	Name() string

	// Start initializes and connects to the backend server.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the server connection.
	Stop() error

	// DiscoverTools queries the backend for available tools.
	DiscoverTools(ctx context.Context) ([]mcp.Tool, error)

	// CallTool executes a tool on the backend server.
	CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error)

	// IsStarted returns whether the server is currently running.
	IsStarted() bool

	// Config returns the server configuration.
	Config() *config.ServerConfig
}

// ResourceServer is an optional interface for servers that support MCP resources.
type ResourceServer interface {
	Server

	// DiscoverResources queries the backend for available resources.
	DiscoverResources(ctx context.Context) ([]mcp.Resource, error)

	// ReadResource retrieves content from a resource URI.
	ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error)
}

// PromptServer is an optional interface for servers that support MCP prompts.
type PromptServer interface {
	Server

	// DiscoverPrompts queries the backend for available prompts.
	DiscoverPrompts(ctx context.Context) ([]mcp.Prompt, error)

	// GetPrompt retrieves a prompt with the given arguments.
	GetPrompt(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error)
}

// FullServer combines all MCP capabilities - tools, resources, and prompts.
type FullServer interface {
	Server
	ResourceServer
	PromptServer
}

// Ensure ManagedServer implements Server interface.
var _ Server = (*ManagedServer)(nil)
