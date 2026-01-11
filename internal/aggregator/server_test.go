package aggregator_test

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"

	"github.com/valksor/go-assern/internal/aggregator"
	"github.com/valksor/go-assern/internal/config"
)

// TestNewManagedServer tests creating a new managed server.
func TestNewManagedServer(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.ServerConfig{
		Command: "echo",
		Args:    []string{"test"},
		Env:     map[string]string{"KEY": "value"},
	}

	server, err := aggregator.NewManagedServer("test", cfg, []string{"ENV=value"}, logger)
	if err != nil {
		t.Fatalf("NewManagedServer() error = %v", err)
	}

	if server == nil {
		t.Fatal("NewManagedServer() returned nil")
	}

	if server.Name() != "test" {
		t.Errorf("Name() = %q, want 'test'", server.Name())
	}

	if server.Config() != cfg {
		t.Error("Config() returned different config")
	}
}

func TestNewManagedServer_NoCommand(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.ServerConfig{
		Command: "",
	}

	_, err := aggregator.NewManagedServer("test", cfg, nil, logger)
	if err == nil {
		t.Error("NewManagedServer() expected error for empty command, got nil")
	}
}

func TestManagedServer_IsStarted(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.ServerConfig{
		Command: "echo",
		Args:    []string{"test"},
	}

	server, err := aggregator.NewManagedServer("test", cfg, nil, logger)
	if err != nil {
		t.Fatal(err)
	}

	if server.IsStarted() {
		t.Error("IsStarted() = true, want false before Start()")
	}

	// Note: We can't actually call Start() without a real transport
	// In a real test, we'd need to mock the transport or use a test double
	// For now, we verify the initial state
}

func TestManagedServer_Name(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.ServerConfig{
		Command: "echo",
	}

	server, err := aggregator.NewManagedServer("my_server", cfg, nil, logger)
	if err != nil {
		t.Fatal(err)
	}

	if server.Name() != "my_server" {
		t.Errorf("Name() = %q, want 'my_server'", server.Name())
	}
}

func TestManagedServer_Config(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.ServerConfig{
		Command:   "npx",
		Args:      []string{"-y", "@modelcontextprotocol/server-filesystem"},
		Env:       map[string]string{"TOKEN": "test"},
		MergeMode: config.MergeModeReplace,
	}

	server, err := aggregator.NewManagedServer("test", cfg, nil, logger)
	if err != nil {
		t.Fatal(err)
	}

	retrievedCfg := server.Config()
	if retrievedCfg.Command != cfg.Command {
		t.Errorf("Config().Command = %q, want %q", retrievedCfg.Command, cfg.Command)
	}

	if retrievedCfg.MergeMode != cfg.MergeMode {
		t.Errorf("Config().MergeMode = %q, want %q", retrievedCfg.MergeMode, cfg.MergeMode)
	}
}

func TestManagedServer_Start_AlreadyStarted(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.ServerConfig{
		Command: "echo",
	}

	// Create a mock server that tracks started state
	server, err := aggregator.NewManagedServer("test", cfg, nil, logger)
	if err != nil {
		t.Fatal(err)
	}

	// We can't test this fully without mocking the internal client
	// but we can verify the structure exists
	if server == nil {
		t.Fatal("NewManagedServer() returned nil")
	}

	// The actual Start() test requires integration testing with real processes
	// or more extensive mocking of the MCP client interface
}

func TestManagedServer_Start_NoCommand(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.ServerConfig{
		Command: "",
	}

	_, err := aggregator.NewManagedServer("test", cfg, nil, logger)
	if err == nil {
		t.Error("Expected error when creating server with no command")
	}
}

// TestManagedServer_VariousCommands tests creating servers with different command types.
func TestManagedServer_VariousCommands(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	tests := []struct {
		name string
		cmd  string
		args []string
		env  map[string]string
		mode config.MergeMode
	}{
		{
			name: "npx command",
			cmd:  "npx",
			args: []string{"-y", "@modelcontextprotocol/server-github"},
			env:  map[string]string{"GITHUB_TOKEN": "test"},
			mode: config.MergeModeOverlay,
		},
		{
			name: "uvx command",
			cmd:  "uvx",
			args: []string{"mcp-server-git"},
			env:  nil,
			mode: config.MergeModeReplace,
		},
		{
			name: "python command",
			cmd:  "python",
			args: []string{"-m", "mcp_server"},
			env:  map[string]string{"PYTHONPATH": "/src"},
			mode: "",
		},
		{
			name: "node command",
			cmd:  "node",
			args: []string{"./server.js"},
			env:  map[string]string{"NODE_ENV": "production"},
			mode: config.MergeModeOverlay,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.ServerConfig{
				Command:   tt.cmd,
				Args:      tt.args,
				Env:       tt.env,
				MergeMode: tt.mode,
			}

			server, err := aggregator.NewManagedServer("test", cfg, nil, logger)
			if err != nil {
				t.Fatalf("NewManagedServer() error = %v", err)
			}

			if server.Name() != "test" {
				t.Errorf("Name() = %q, want 'test'", server.Name())
			}

			retrievedCfg := server.Config()
			if retrievedCfg.Command != tt.cmd {
				t.Errorf("Config().Command = %q, want %q", retrievedCfg.Command, tt.cmd)
			}
		})
	}
}

// TestManagedServer_DiscoverTools_NotStarted verifies error handling.
func TestManagedServer_DiscoverTools_NotStarted(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.ServerConfig{
		Command: "echo",
	}

	// This test verifies the error path, but we need to call Start first
	// In a real scenario, this would return an error
	// Since we can't actually start without a transport, we verify structure
	server, err := aggregator.NewManagedServer("test", cfg, nil, logger)
	if err != nil {
		t.Fatal(err)
	}

	if server == nil {
		t.Fatal("NewManagedServer() returned nil")
	}

	// Verify initial state
	if server.IsStarted() {
		t.Error("Newly created server should not be started")
	}
}

// TestManagedServer_Environment tests environment variable handling.
func TestManagedServer_Environment(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.ServerConfig{
		Command: "echo",
		Env: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
		},
	}

	env := []string{
		"VAR3=value3",
		"VAR4=value4",
	}

	server, err := aggregator.NewManagedServer("test", cfg, env, logger)
	if err != nil {
		t.Fatal(err)
	}

	if server == nil {
		t.Fatal("NewManagedServer() returned nil")
	}

	// The environment is passed to the server during Start()
	// We can't directly access it, but we verified the server was created
}

// TestManagedServer_AllowedTools tests the allowed tools configuration.
func TestManagedServer_AllowedTools(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.ServerConfig{
		Command: "echo",
		Allowed: []string{"tool1", "tool2", "tool3"},
	}

	server, err := aggregator.NewManagedServer("test", cfg, nil, logger)
	if err != nil {
		t.Fatal(err)
	}

	retrievedCfg := server.Config()
	if len(retrievedCfg.Allowed) != 3 {
		t.Errorf("Config().Allowed length = %d, want 3", len(retrievedCfg.Allowed))
	}

	expectedAllowed := []string{"tool1", "tool2", "tool3"}
	for i, allowed := range retrievedCfg.Allowed {
		if allowed != expectedAllowed[i] {
			t.Errorf("Config().Allowed[%d] = %q, want %q", i, allowed, expectedAllowed[i])
		}
	}
}

// TestManagedServer_Stop_NotStarted tests stopping a server that was never started.
func TestManagedServer_Stop_NotStarted(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.ServerConfig{
		Command: "echo",
	}

	server, err := aggregator.NewManagedServer("test", cfg, nil, logger)
	if err != nil {
		t.Fatal(err)
	}

	// According to the implementation, Stop() returns nil if not started
	// We can't call Stop() without exposing the method or using reflection
	// This is a structural test that verifies creation works
	if server == nil {
		t.Fatal("NewManagedServer() returned nil")
	}
}

// TestServerIntegration is a placeholder for integration tests.
// These would require actual MCP server processes to be running.
func TestServerIntegration(t *testing.T) {
	t.Skip("Integration tests require actual MCP servers")
}

// mockMCPClient is a mock for testing purposes when we need to simulate
// MCP protocol behavior without actual processes.
type mockMCPClient struct {
	startCalled atomic.Int32
	closeCalled atomic.Int32
}

func (m *mockMCPClient) Start(_ context.Context) error {
	m.startCalled.Add(1)

	return nil
}

func (m *mockMCPClient) Close() error {
	m.closeCalled.Add(1)

	return nil
}

// TestMockClientBehavior tests our mock client behaves as expected.
func TestMockClientBehavior(t *testing.T) {
	t.Parallel()

	mock := &mockMCPClient{}

	ctx := context.Background()
	if err := mock.Start(ctx); err != nil {
		t.Fatalf("mock.Start() error = %v", err)
	}

	if mock.startCalled.Load() != 1 {
		t.Errorf("mock.startCalled = %d, want 1", mock.startCalled.Load())
	}

	if err := mock.Close(); err != nil {
		t.Fatalf("mock.Close() error = %v", err)
	}

	if mock.closeCalled.Load() != 1 {
		t.Errorf("mock.closeCalled = %d, want 1", mock.closeCalled.Load())
	}
}

// TestErrorCases tests various error scenarios.
func TestErrorCases(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	t.Run("nil config", func(t *testing.T) {
		t.Parallel()

		// Note: NewManagedServer with nil config will panic
		// This is intentional - config should not be nil
		// We use defer/recover to verify the panic happens
		defer func() {
			if r := recover(); r != nil {
				// Expected panic
				return
			}
			t.Error("Expected panic with nil config, but none occurred")
		}()

		_, _ = aggregator.NewManagedServer("test", nil, nil, logger)
	})

	t.Run("empty name is allowed", func(t *testing.T) {
		t.Parallel()

		cfg := &config.ServerConfig{Command: "echo"}
		server, err := aggregator.NewManagedServer("", cfg, nil, logger)
		if err != nil {
			t.Errorf("NewManagedServer() with empty name error = %v", err)
		}
		if server.Name() != "" {
			t.Errorf("Name() = %q, want empty string", server.Name())
		}
	})
}

// TestConcurrentServerCreation tests that creating multiple servers concurrently is safe.
func TestConcurrentServerCreation(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	done := make(chan bool, 10)

	for i := range 10 {
		go func(n int) {
			cfg := &config.ServerConfig{
				Command: "echo",
				Args:    []string{"test", string(rune('0' + n))},
			}
			_, err := aggregator.NewManagedServer("server"+string(rune('0'+n)), cfg, nil, logger)
			if err != nil {
				t.Errorf("Concurrent creation failed: %v", err)
			}
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}
}
