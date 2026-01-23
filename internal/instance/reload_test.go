package instance

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/valksor/go-assern/internal/aggregator"
	"github.com/valksor/go-assern/internal/config"
)

func TestReload_Success(t *testing.T) {
	// Create temp socket
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Set up HOME for config loading
	globalDir := tmpDir + "/.valksor/assern"
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatalf("failed to create global dir: %v", err)
	}
	mcpJSON := `{"mcpServers": {}}`
	if err := os.WriteFile(globalDir+"/mcp.json", []byte(mcpJSON), 0o644); err != nil {
		t.Fatalf("failed to write mcp.json: %v", err)
	}

	// t.Setenv restores automatically after test
	t.Setenv("HOME", tmpDir)

	// Create aggregator
	cfg := &config.Config{
		Servers:  map[string]*config.ServerConfig{},
		Settings: config.DefaultSettings(),
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	agg, err := aggregator.New(aggregator.Options{
		Config:      cfg,
		Logger:      logger,
		WorkDir:     tmpDir,
		ProjectName: "",
	})
	if err != nil {
		t.Fatalf("failed to create aggregator: %v", err)
	}

	// Create MCP server
	mcpServer := server.NewMCPServer("test", "1.0.0")

	// Start socket server
	sockServer := NewServer(socketPath, mcpServer, agg, logger)
	if err := sockServer.Start(); err != nil {
		t.Fatalf("failed to start socket server: %v", err)
	}
	defer func() { _ = sockServer.Stop() }()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Call Reload
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := Reload(ctx, socketPath)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// No servers configured, so no changes
	if result.Added != 0 {
		t.Errorf("expected 0 added, got %d", result.Added)
	}
	if result.Removed != 0 {
		t.Errorf("expected 0 removed, got %d", result.Removed)
	}
}

func TestReload_NoSocket(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := Reload(ctx, "/nonexistent/socket.sock")
	if err == nil {
		t.Error("expected error for non-existent socket")
	}
}

func TestReload_InvalidResponse(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns invalid response
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Start a simple listener that returns garbage
	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		// Read request
		buf := make([]byte, 1024)
		_, _ = conn.Read(buf)

		// Send invalid JSON
		_, _ = conn.Write([]byte("not json\n"))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = Reload(ctx, socketPath)
	if err == nil {
		t.Error("expected error for invalid response")
	}
}

func TestReload_ErrorResponse(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns an error response
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		// Read request
		buf := make([]byte, 1024)
		_, _ = conn.Read(buf)

		// Send error response
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]any{
				"code":    -32603,
				"message": "test error",
			},
		}
		data, err := json.Marshal(resp)
		if err != nil {
			return
		}
		_, _ = conn.Write(append(data, '\n'))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = Reload(ctx, socketPath)
	if err == nil {
		t.Error("expected error for error response")
	}
	if err.Error() != "reload error: test error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReloadResult_Fields(t *testing.T) {
	t.Parallel()

	result := ReloadResult{
		Added:   5,
		Removed: 3,
		Errors:  []string{"err1", "err2"},
	}

	if result.Added != 5 {
		t.Errorf("expected Added=5, got %d", result.Added)
	}
	if result.Removed != 3 {
		t.Errorf("expected Removed=3, got %d", result.Removed)
	}
	if len(result.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors))
	}
}
