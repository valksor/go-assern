package instance

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	socketPath := "/tmp/test.sock"
	client := NewClient(socketPath)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.socketPath != socketPath {
		t.Errorf("NewClient() socketPath = %s, want %s", client.socketPath, socketPath)
	}

	if client.requestID != 0 {
		t.Errorf("NewClient() requestID = %d, want 0", client.requestID)
	}
}

func TestClient_QueryTools(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create MCP server with test tools
	mcpServer := server.NewMCPServer("test-server", "1.0.0")
	mcpServer.AddTool(
		mcp.NewTool("test_tool_one", mcp.WithDescription("First test tool")),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("one"), nil
		},
	)
	mcpServer.AddTool(
		mcp.NewTool("test_tool_two", mcp.WithDescription("Second test tool")),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("two"), nil
		},
	)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	srv := NewServer(socketPath, mcpServer, nil, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Query tools
	ctx := t.Context()
	result, err := QueryTools(ctx, socketPath)
	if err != nil {
		t.Fatalf("QueryTools() error = %v", err)
	}

	if result == nil {
		t.Fatal("QueryTools() returned nil result")
	}

	if len(result.Tools) != 2 {
		t.Fatalf("Expected 2 tools, got %d", len(result.Tools))
	}

	// Check tool names (order may vary)
	toolNames := make(map[string]string)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = tool.Description
	}

	if desc, ok := toolNames["test_tool_one"]; !ok {
		t.Error("Missing test_tool_one")
	} else if desc != "First test tool" {
		t.Errorf("test_tool_one description = %s, want 'First test tool'", desc)
	}

	if desc, ok := toolNames["test_tool_two"]; !ok {
		t.Error("Missing test_tool_two")
	} else if desc != "Second test tool" {
		t.Errorf("test_tool_two description = %s, want 'Second test tool'", desc)
	}
}

func TestClient_QueryTools_NoServer(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "nonexistent.sock")

	ctx := t.Context()
	_, err := QueryTools(ctx, socketPath)

	if err == nil {
		t.Fatal("QueryTools() should return error for non-existent socket")
	}
}

func TestClient_ConnectClose(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	srv := NewServer(socketPath, mcpServer, nil, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	client := NewClient(socketPath)

	ctx := t.Context()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if client.conn == nil {
		t.Error("Connect() did not set conn")
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestClient_CloseWithoutConnect(t *testing.T) {
	t.Parallel()

	client := NewClient("/tmp/test.sock")

	// Close without connect should not error
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestClient_QueryTools_ManyTools(t *testing.T) {
	// Use short path to avoid Unix socket path length limits on macOS
	socketPath := filepath.Join("/tmp", fmt.Sprintf("assern-test-%d", os.Getpid()), "many.sock")
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		t.Fatalf("mkdir error = %v", err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(socketPath)) }()

	// Create MCP server with many tools
	mcpServer := server.NewMCPServer("many-tools-server", "1.0.0")
	const numTools = 50

	for i := range numTools {
		name := fmt.Sprintf("tool_%d", i)
		desc := fmt.Sprintf("Tool number %d", i)
		mcpServer.AddTool(
			mcp.NewTool(name, mcp.WithDescription(desc)),
			func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return mcp.NewToolResultText("ok"), nil
			},
		)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	srv := NewServer(socketPath, mcpServer, nil, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	ctx := t.Context()
	result, err := QueryTools(ctx, socketPath)
	if err != nil {
		t.Fatalf("QueryTools() error = %v", err)
	}

	if len(result.Tools) != numTools {
		t.Errorf("Expected %d tools, got %d", numTools, len(result.Tools))
	}
}

func TestClient_MultipleQueries(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test-server", "1.0.0")
	mcpServer.AddTool(
		mcp.NewTool("test_tool", mcp.WithDescription("A test tool")),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("ok"), nil
		},
	)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	srv := NewServer(socketPath, mcpServer, nil, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Make multiple sequential queries
	for i := range 3 {
		ctx := t.Context()
		result, err := QueryTools(ctx, socketPath)
		if err != nil {
			t.Fatalf("QueryTools() iteration %d error = %v", i, err)
		}

		if len(result.Tools) != 1 {
			t.Errorf("Iteration %d: Expected 1 tool, got %d", i, len(result.Tools))
		}
	}
}

func TestClient_Initialize(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test-server", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	srv := NewServer(socketPath, mcpServer, nil, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	client := NewClient(socketPath)

	ctx := t.Context()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// After Initialize, requestID should have incremented
	if client.requestID != 1 {
		t.Errorf("requestID after Initialize = %d, want 1", client.requestID)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "nonexistent.sock")

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := QueryTools(ctx, socketPath)
	if err == nil {
		t.Fatal("QueryTools() should return error with cancelled context")
	}
}

func TestListResult_Empty(t *testing.T) {
	t.Parallel()

	result := &ListResult{
		Tools: []ToolInfo{},
	}

	if len(result.Tools) != 0 {
		t.Errorf("Expected empty tools, got %d", len(result.Tools))
	}
}

func TestToolInfo_Fields(t *testing.T) {
	t.Parallel()

	tool := ToolInfo{
		Name:        "test_tool",
		Description: "A test tool description",
	}

	if tool.Name != "test_tool" {
		t.Errorf("Name = %s, want test_tool", tool.Name)
	}

	if tool.Description != "A test tool description" {
		t.Errorf("Description = %s, want 'A test tool description'", tool.Description)
	}
}
