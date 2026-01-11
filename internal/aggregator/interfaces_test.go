package aggregator

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/valksor/go-assern/internal/config"
)

// mockServer is a test implementation of the Server interface.
type mockServer struct {
	name      string
	cfg       *config.ServerConfig
	started   bool
	tools     []mcp.Tool
	startErr  error
	stopErr   error
	toolsErr  error
	callErr   error
	callCount int
}

func (m *mockServer) Name() string {
	return m.name
}

func (m *mockServer) Start(ctx context.Context) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.started = true

	return nil
}

func (m *mockServer) Stop() error {
	if m.stopErr != nil {
		return m.stopErr
	}
	m.started = false

	return nil
}

func (m *mockServer) DiscoverTools(ctx context.Context) ([]mcp.Tool, error) {
	if m.toolsErr != nil {
		return nil, m.toolsErr
	}

	return m.tools, nil
}

func (m *mockServer) CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	m.callCount++
	if m.callErr != nil {
		return nil, m.callErr
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: "mock result"},
		},
	}, nil
}

func (m *mockServer) IsStarted() bool {
	return m.started
}

func (m *mockServer) Config() *config.ServerConfig {
	return m.cfg
}

// TestServerInterface ensures ManagedServer implements the Server interface.
func TestServerInterface(t *testing.T) {
	t.Parallel()

	// This is a compile-time check - if ManagedServer doesn't implement Server,
	// the code won't compile. The var _ Server = (*ManagedServer)(nil) in
	// interfaces.go does the same, but this test documents the expectation.
	var _ Server = (*ManagedServer)(nil)
}

// TestMockServerImplementsInterface verifies our test mock implements Server.
func TestMockServerImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ Server = (*mockServer)(nil)
}

func TestMockServer_Lifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mock := &mockServer{
		name: "test-server",
		cfg: &config.ServerConfig{
			Command: "echo",
			Args:    []string{"hello"},
		},
		tools: []mcp.Tool{
			{Name: "test_tool", Description: "A test tool"},
		},
	}

	// Test Name
	if mock.Name() != "test-server" {
		t.Errorf("Name() = %q, want %q", mock.Name(), "test-server")
	}

	// Test IsStarted before Start
	if mock.IsStarted() {
		t.Error("IsStarted() = true before Start(), want false")
	}

	// Test Start
	if err := mock.Start(ctx); err != nil {
		t.Errorf("Start() error = %v", err)
	}

	if !mock.IsStarted() {
		t.Error("IsStarted() = false after Start(), want true")
	}

	// Test DiscoverTools
	tools, err := mock.DiscoverTools(ctx)
	if err != nil {
		t.Errorf("DiscoverTools() error = %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("DiscoverTools() returned %d tools, want 1", len(tools))
	}

	// Test CallTool
	result, err := mock.CallTool(ctx, "test_tool", nil)
	if err != nil {
		t.Errorf("CallTool() error = %v", err)
	}

	if result == nil {
		t.Error("CallTool() returned nil result")
	}

	if mock.callCount != 1 {
		t.Errorf("callCount = %d, want 1", mock.callCount)
	}

	// Test Config
	cfg := mock.Config()
	if cfg.Command != "echo" {
		t.Errorf("Config().Command = %q, want %q", cfg.Command, "echo")
	}

	// Test Stop
	if err := mock.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	if mock.IsStarted() {
		t.Error("IsStarted() = true after Stop(), want false")
	}
}

func TestMockServer_Errors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("start error", func(t *testing.T) {
		t.Parallel()

		mock := &mockServer{
			name:     "error-server",
			startErr: ErrInvalidTransport,
		}

		err := mock.Start(ctx)
		if !errors.Is(err, ErrInvalidTransport) {
			t.Errorf("Start() error = %v, want %v", err, ErrInvalidTransport)
		}
	})

	t.Run("tools error", func(t *testing.T) {
		t.Parallel()

		mock := &mockServer{
			name:     "tools-error-server",
			toolsErr: ErrServerNotStarted,
		}

		_, err := mock.DiscoverTools(ctx)
		if !errors.Is(err, ErrServerNotStarted) {
			t.Errorf("DiscoverTools() error = %v, want %v", err, ErrServerNotStarted)
		}
	})

	t.Run("call error", func(t *testing.T) {
		t.Parallel()

		mock := &mockServer{
			name:    "call-error-server",
			callErr: ErrServerNotFound,
		}

		_, err := mock.CallTool(ctx, "test", nil)
		if !errors.Is(err, ErrServerNotFound) {
			t.Errorf("CallTool() error = %v, want %v", err, ErrServerNotFound)
		}
	})
}
