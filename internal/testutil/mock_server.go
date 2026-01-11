// Package testutil provides testing utilities for the assern project.
package testutil

import (
	"context"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/valksor/go-assern/internal/config"
)

// MockServer is a test implementation of the aggregator.Server interface.
// It allows tests to inject controlled behavior without spawning real processes.
type MockServer struct {
	ServerName string
	ServerCfg  *config.ServerConfig
	Tools      []mcp.Tool
	Resources  []mcp.Resource
	Prompts    []mcp.Prompt

	// Configurable errors for testing error paths
	StartErr     error
	StopErr      error
	ToolsErr     error
	CallErr      error
	ResourcesErr error
	PromptsErr   error

	// Configurable responses
	ToolResults map[string]*mcp.CallToolResult

	// Call tracking
	mu            sync.RWMutex
	started       bool
	ToolCalls     []ToolCallRecord
	ResourceReads []ResourceReadRecord
	PromptGets    []PromptGetRecord
}

// ToolCallRecord tracks a tool call for verification.
type ToolCallRecord struct {
	Name string
	Args map[string]any
}

// ResourceReadRecord tracks a resource read for verification.
type ResourceReadRecord struct {
	URI string
}

// PromptGetRecord tracks a prompt get for verification.
type PromptGetRecord struct {
	Name string
	Args map[string]string
}

// NewMockServer creates a new mock server with the given name and tools.
func NewMockServer(name string, tools []mcp.Tool) *MockServer {
	return &MockServer{
		ServerName:  name,
		ServerCfg:   &config.ServerConfig{Command: "mock"},
		Tools:       tools,
		ToolResults: make(map[string]*mcp.CallToolResult),
	}
}

// Name returns the server identifier.
func (m *MockServer) Name() string {
	return m.ServerName
}

// Start initializes the mock server.
func (m *MockServer) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.StartErr != nil {
		return m.StartErr
	}

	m.started = true

	return nil
}

// Stop shuts down the mock server.
func (m *MockServer) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.StopErr != nil {
		return m.StopErr
	}

	m.started = false

	return nil
}

// DiscoverTools returns the configured tools.
func (m *MockServer) DiscoverTools(ctx context.Context) ([]mcp.Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ToolsErr != nil {
		return nil, m.ToolsErr
	}

	return m.Tools, nil
}

// CallTool executes a mock tool call.
func (m *MockServer) CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ToolCalls = append(m.ToolCalls, ToolCallRecord{Name: name, Args: args})

	if m.CallErr != nil {
		return nil, m.CallErr
	}

	// Return configured result if available
	if result, ok := m.ToolResults[name]; ok {
		return result, nil
	}

	// Default result
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: "mock result for " + name},
		},
	}, nil
}

// IsStarted returns whether the server is running.
func (m *MockServer) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.started
}

// Config returns the server configuration.
func (m *MockServer) Config() *config.ServerConfig {
	return m.ServerCfg
}

// DiscoverResources returns the configured resources.
func (m *MockServer) DiscoverResources(ctx context.Context) ([]mcp.Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ResourcesErr != nil {
		return nil, m.ResourcesErr
	}

	return m.Resources, nil
}

// ReadResource reads a mock resource.
func (m *MockServer) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ResourceReads = append(m.ResourceReads, ResourceReadRecord{URI: uri})

	if m.ResourcesErr != nil {
		return nil, m.ResourcesErr
	}

	return &mcp.ReadResourceResult{
		Contents: []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "text/plain",
				Text:     "mock content for " + uri,
			},
		},
	}, nil
}

// DiscoverPrompts returns the configured prompts.
func (m *MockServer) DiscoverPrompts(ctx context.Context) ([]mcp.Prompt, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.PromptsErr != nil {
		return nil, m.PromptsErr
	}

	return m.Prompts, nil
}

// GetPrompt retrieves a mock prompt.
func (m *MockServer) GetPrompt(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PromptGets = append(m.PromptGets, PromptGetRecord{Name: name, Args: args})

	if m.PromptsErr != nil {
		return nil, m.PromptsErr
	}

	return &mcp.GetPromptResult{
		Description: "Mock prompt: " + name,
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: "mock prompt content for " + name,
				},
			},
		},
	}, nil
}

// SetToolResult configures the result for a specific tool.
func (m *MockServer) SetToolResult(toolName string, result *mcp.CallToolResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ToolResults[toolName] = result
}

// GetToolCalls returns all recorded tool calls.
func (m *MockServer) GetToolCalls() []ToolCallRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	calls := make([]ToolCallRecord, len(m.ToolCalls))
	copy(calls, m.ToolCalls)

	return calls
}

// Reset clears all recorded calls.
func (m *MockServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ToolCalls = nil
	m.ResourceReads = nil
	m.PromptGets = nil
}
