package aggregator_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/valksor/go-assern/internal/aggregator"
	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/testutil"
)

func TestAggregator_Reload_NoChanges(t *testing.T) {
	// Create temp dir for config
	tmpDir := t.TempDir()
	globalDir := tmpDir + "/.valksor/assern"
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatalf("failed to create global dir: %v", err)
	}

	// Write initial config
	initialMCP := `{"mcpServers": {}}`
	if err := os.WriteFile(globalDir+"/mcp.json", []byte(initialMCP), 0o644); err != nil {
		t.Fatalf("failed to write mcp.json: %v", err)
	}

	// Set HOME to temp dir (t.Setenv restores automatically)
	t.Setenv("HOME", tmpDir)

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

	// Reload without any config changes - should return empty result
	result, err := agg.Reload(context.Background())
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if result.Added != 0 {
		t.Errorf("expected 0 added, got %d", result.Added)
	}
	if result.Removed != 0 {
		t.Errorf("expected 0 removed, got %d", result.Removed)
	}
}

func TestAggregator_Reload_AddMultipleServers(t *testing.T) {
	t.Parallel()

	// Create mock servers with multiple tools each
	mockServer1 := testutil.NewMockServer("mock1", []mcp.Tool{
		{Name: "tool1a", Description: "Test tool 1a"},
		{Name: "tool1b", Description: "Test tool 1b"},
	})
	mockServer2 := testutil.NewMockServer("mock2", []mcp.Tool{
		{Name: "tool2a", Description: "Test tool 2a"},
	})

	cfg := &config.Config{
		Servers:  map[string]*config.ServerConfig{},
		Settings: config.DefaultSettings(),
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	agg, err := aggregator.New(aggregator.Options{
		Config:      cfg,
		Logger:      logger,
		WorkDir:     t.TempDir(),
		ProjectName: "",
	})
	if err != nil {
		t.Fatalf("failed to create aggregator: %v", err)
	}

	// Start with no servers
	names := agg.ServerNames()
	if len(names) != 0 {
		t.Errorf("expected 0 servers initially, got %d", len(names))
	}

	// Add first mock server
	ctx := context.Background()
	if err := agg.AddServer(ctx, mockServer1); err != nil {
		t.Fatalf("failed to add mock server 1: %v", err)
	}

	// Verify first server added
	names = agg.ServerNames()
	if len(names) != 1 {
		t.Errorf("expected 1 server after first add, got %d", len(names))
	}

	// Verify tools from first server
	tools := agg.ListTools()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools after first add, got %d", len(tools))
	}

	// Add second mock server
	if err := agg.AddServer(ctx, mockServer2); err != nil {
		t.Fatalf("failed to add mock server 2: %v", err)
	}

	// Verify both servers added
	names = agg.ServerNames()
	if len(names) != 2 {
		t.Errorf("expected 2 servers after second add, got %d", len(names))
	}

	// Verify all tools registered (2 from mock1 + 1 from mock2 = 3)
	tools = agg.ListTools()
	if len(tools) != 3 {
		t.Errorf("expected 3 total tools, got %d", len(tools))
	}
}

func TestAggregator_Reload_WithMockServer(t *testing.T) {
	t.Parallel()

	// Create mock server with mcp.Tool
	mockServer := testutil.NewMockServer("mock1", []mcp.Tool{
		{Name: "tool1", Description: "Test tool 1"},
	})

	cfg := &config.Config{
		Servers:  map[string]*config.ServerConfig{},
		Settings: config.DefaultSettings(),
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	agg, err := aggregator.New(aggregator.Options{
		Config:      cfg,
		Logger:      logger,
		WorkDir:     t.TempDir(),
		ProjectName: "",
	})
	if err != nil {
		t.Fatalf("failed to create aggregator: %v", err)
	}

	// Add mock server
	ctx := context.Background()
	if err := agg.AddServer(ctx, mockServer); err != nil {
		t.Fatalf("failed to add mock server: %v", err)
	}

	// Verify server was added
	names := agg.ServerNames()
	if len(names) != 1 {
		t.Errorf("expected 1 server, got %d", len(names))
	}
	if names[0] != "mock1" {
		t.Errorf("expected server name 'mock1', got '%s'", names[0])
	}

	// Verify tools were registered
	tools := agg.ListTools()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}
}

func TestReloadResult_Fields(t *testing.T) {
	t.Parallel()

	result := aggregator.ReloadResult{
		Added:   2,
		Removed: 1,
		Errors:  []string{"error1", "error2"},
	}

	if result.Added != 2 {
		t.Errorf("expected Added=2, got %d", result.Added)
	}
	if result.Removed != 1 {
		t.Errorf("expected Removed=1, got %d", result.Removed)
	}
	if len(result.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors))
	}
}
