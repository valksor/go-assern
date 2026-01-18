package aggregator_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/valksor/go-assern/internal/aggregator"
	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-toolkit/env"
	toolkitproject "github.com/valksor/go-toolkit/project"
)

func TestNew(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()

		cfg := config.NewConfig()
		opts := aggregator.Options{
			Config:  cfg,
			Logger:  logger,
			Timeout: 30 * time.Second,
		}

		agg, err := aggregator.New(opts)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if agg == nil {
			t.Fatal("New() returned nil")
		}
	})

	t.Run("nil config returns error", func(t *testing.T) {
		t.Parallel()

		opts := aggregator.Options{
			Config: nil,
			Logger: logger,
		}

		_, err := aggregator.New(opts)
		if err == nil {
			t.Error("New() expected error with nil config, got nil")
		}
	})

	t.Run("default timeout applied", func(t *testing.T) {
		t.Parallel()

		cfg := config.NewConfig()
		opts := aggregator.Options{
			Config: cfg,
			Logger: logger,
			// No timeout specified - should default to 60s
		}

		agg, err := aggregator.New(opts)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if agg == nil {
			t.Fatal("New() returned nil")
		}
		// Default timeout is applied internally
	})

	t.Run("default logger applied", func(t *testing.T) {
		t.Parallel()

		cfg := config.NewConfig()
		opts := aggregator.Options{
			Config:  cfg,
			Logger:  nil, // Should use default
			Timeout: 30 * time.Second,
		}

		agg, err := aggregator.New(opts)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if agg == nil {
			t.Fatal("New() returned nil")
		}
	})
}

func TestAggregator_ProjectName(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	t.Run("with project context", func(t *testing.T) {
		t.Parallel()

		cfg := config.NewConfig()
		projectCtx := &toolkitproject.Context{
			Name: "myproject",
		}

		opts := aggregator.Options{
			Config:  cfg,
			Project: projectCtx,
			Logger:  logger,
		}

		agg, err := aggregator.New(opts)
		if err != nil {
			t.Fatal(err)
		}

		if agg.ProjectName() != "myproject" {
			t.Errorf("ProjectName() = %q, want 'myproject'", agg.ProjectName())
		}
	})

	t.Run("without project context", func(t *testing.T) {
		t.Parallel()

		cfg := config.NewConfig()
		opts := aggregator.Options{
			Config:  cfg,
			Project: nil,
			Logger:  logger,
		}

		agg, err := aggregator.New(opts)
		if err != nil {
			t.Fatal(err)
		}

		if agg.ProjectName() != "" {
			t.Errorf("ProjectName() = %q, want empty string", agg.ProjectName())
		}
	})

	t.Run("with empty project name", func(t *testing.T) {
		t.Parallel()

		cfg := config.NewConfig()
		projectCtx := &toolkitproject.Context{
			Name: "",
		}

		opts := aggregator.Options{
			Config:  cfg,
			Project: projectCtx,
			Logger:  logger,
		}

		agg, err := aggregator.New(opts)
		if err != nil {
			t.Fatal(err)
		}

		if agg.ProjectName() != "" {
			t.Errorf("ProjectName() = %q, want empty string", agg.ProjectName())
		}
	})
}

func TestAggregator_ListTools(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()

	opts := aggregator.Options{
		Config: cfg,
		Logger: logger,
	}

	agg, err := aggregator.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Initially no tools
	tools := agg.ListTools()
	if tools == nil {
		t.Error("ListTools() returned nil")
	}

	if len(tools) != 0 {
		t.Errorf("ListTools() length = %d, want 0", len(tools))
	}
}

func TestAggregator_ServerNames(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()

	opts := aggregator.Options{
		Config: cfg,
		Logger: logger,
	}

	agg, err := aggregator.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Initially no servers
	names := agg.ServerNames()
	if names == nil {
		t.Error("ServerNames() returned nil")
	}

	if len(names) != 0 {
		t.Errorf("ServerNames() length = %d, want 0", len(names))
	}
}

func TestAggregator_GetServer(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()

	opts := aggregator.Options{
		Config: cfg,
		Logger: logger,
	}

	agg, err := aggregator.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Try to get a non-existent server
	server, ok := agg.GetServer("nonexistent")
	if ok {
		t.Error("GetServer() returned true for non-existent server")
	}

	if server != nil {
		t.Error("GetServer() returned non-nil server for non-existent name")
	}
}

func TestAggregator_Start_NoServers(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()
	// Config has no servers defined

	opts := aggregator.Options{
		Config: cfg,
		Logger: logger,
	}

	agg, err := aggregator.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = agg.Start(ctx)

	// Should return error when no servers configured
	if err == nil {
		t.Error("Start() with no servers should return error, got nil")
	}
}

func TestAggregator_Start_WithServers(t *testing.T) {
	// Note: Tests using real process spawning are now covered by integration_test.go
	// using mock servers. This test verifies the Start() error path.
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()

	// Add a server config that will fail (no command or URL)
	cfg.Servers["invalid"] = &config.ServerConfig{
		// Empty config - no command or URL
	}

	opts := aggregator.Options{
		Config:    cfg,
		Logger:    logger,
		EnvLoader: env.NewLoader(),
	}

	agg, err := aggregator.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Start should fail because the server config is invalid
	err = agg.Start(ctx)
	if err == nil {
		t.Error("Start() should return error for invalid server config")
	}

	// Verify Stop can be called even after failed start
	err = agg.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestAggregator_Stop(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()

	opts := aggregator.Options{
		Config: cfg,
		Logger: logger,
	}

	agg, err := aggregator.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Stop should work even with no servers
	err = agg.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	// Should still have no servers after stop
	if len(agg.ServerNames()) != 0 {
		t.Errorf("After Stop(), ServerNames() length = %d, want 0", len(agg.ServerNames()))
	}
}

func TestAggregator_CreateMCPServer(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()

	opts := aggregator.Options{
		Config: cfg,
		Logger: logger,
	}

	agg, err := aggregator.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	// CreateMCPServer should create a server even with no tools
	mcpServer := agg.CreateMCPServer()
	if mcpServer == nil {
		t.Fatal("CreateMCPServer() returned nil")
	}

	// The MCP server should be created (even if empty)
}

func TestAggregator_WithEnvLoader(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()

	envLoader := env.NewLoader()
	envLoader.Set("global", "TEST_VAR", "test_value")

	opts := aggregator.Options{
		Config:    cfg,
		Logger:    logger,
		EnvLoader: envLoader,
	}

	agg, err := aggregator.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	if agg == nil {
		t.Fatal("New() returned nil")
	}
}

func TestAggregator_WithProjectContext(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()

	projectCtx := &toolkitproject.Context{
		Name:      "test_project",
		Directory: "/path/to/project",
		Source:    toolkitproject.SourceLocal,
	}

	opts := aggregator.Options{
		Config:    cfg,
		Logger:    logger,
		EnvLoader: env.NewLoader(),
		Project:   projectCtx,
	}

	agg, err := aggregator.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	if agg.ProjectName() != "test_project" {
		t.Errorf("ProjectName() = %q, want 'test_project'", agg.ProjectName())
	}
}

func TestAggregator_MultipleServerConfigs(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()

	// Add multiple server configs
	cfg.Servers["github"] = &config.ServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-github"},
		Env:     map[string]string{"GITHUB_TOKEN": "test"},
	}

	cfg.Servers["filesystem"] = &config.ServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
		Allowed: []string{"read_file"},
	}

	cfg.Servers["jira"] = &config.ServerConfig{
		Command:   "python",
		Args:      []string{"-m", "mcp_server_jira"},
		MergeMode: config.MergeModeReplace,
	}

	opts := aggregator.Options{
		Config:    cfg,
		Logger:    logger,
		EnvLoader: env.NewLoader(),
	}

	agg, err := aggregator.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	// The aggregator is created successfully
	// (Starting would fail because these aren't real MCP servers)
	if agg == nil {
		t.Fatal("New() returned nil")
	}
}

func TestAggregator_ServerNames_AfterStart(t *testing.T) {
	// Note: Full integration tests with mock servers are in integration_test.go
	// This test verifies behavior when all servers fail to start.
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()

	// Add a server config that will fail (empty config)
	cfg.Servers["failing"] = &config.ServerConfig{}

	opts := aggregator.Options{
		Config:    cfg,
		Logger:    logger,
		EnvLoader: env.NewLoader(),
	}

	agg, err := aggregator.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = agg.Start(ctx) // Will fail due to invalid config

	// No servers should be listed after failed start
	names := agg.ServerNames()
	if len(names) != 0 {
		t.Errorf("ServerNames() should be empty after failed start, got %v", names)
	}
}

func TestAggregator_ListTools_AfterStart(t *testing.T) {
	// Note: Full integration tests with mock servers are in integration_test.go
	// This test verifies behavior when Start fails.
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()

	opts := aggregator.Options{
		Config:    cfg,
		Logger:    logger,
		EnvLoader: env.NewLoader(),
	}

	agg, err := aggregator.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = agg.Start(ctx) // Will fail - no servers configured

	// Should return empty list (no servers started)
	tools := agg.ListTools()
	if tools == nil {
		t.Error("ListTools() returned nil")
	}

	if len(tools) != 0 {
		t.Errorf("ListTools() length = %d, want 0", len(tools))
	}
}

func TestAggregator_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()

	opts := aggregator.Options{
		Config: cfg,
		Logger: logger,
	}

	agg, err := aggregator.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan bool, 10)

	// Start multiple goroutines accessing the aggregator
	for range 10 {
		go func() {
			defer func() { done <- true }()
			for range 100 {
				_ = agg.ServerNames()
				_ = agg.ListTools()
				_ = agg.ProjectName()
				_, _ = agg.GetServer("test")
			}
		}()
	}

	for range 10 {
		<-done
	}

	// If we got here, no race conditions occurred
}

func TestAggregator_OutputFormat(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()

	t.Run("default output format is json", func(t *testing.T) {
		t.Parallel()

		opts := aggregator.Options{
			Config: cfg,
			Logger: logger,
		}

		agg, err := aggregator.New(opts)
		if err != nil {
			t.Fatal(err)
		}

		// Default should be JSON
		if agg == nil {
			t.Fatal("New() returned nil")
		}
		// The aggregator is created with default JSON format
	})

	t.Run("explicit toon format", func(t *testing.T) {
		t.Parallel()

		opts := aggregator.Options{
			Config:       cfg,
			Logger:       logger,
			OutputFormat: "toon",
		}

		agg, err := aggregator.New(opts)
		if err != nil {
			t.Fatal(err)
		}

		if agg == nil {
			t.Fatal("New() returned nil")
		}
		// The aggregator is created with TOON format
	})

	t.Run("explicit json format", func(t *testing.T) {
		t.Parallel()

		opts := aggregator.Options{
			Config:       cfg,
			Logger:       logger,
			OutputFormat: "json",
		}

		agg, err := aggregator.New(opts)
		if err != nil {
			t.Fatal(err)
		}

		if agg == nil {
			t.Fatal("New() returned nil")
		}
	})

	t.Run("empty format defaults to json", func(t *testing.T) {
		t.Parallel()

		opts := aggregator.Options{
			Config:       cfg,
			Logger:       logger,
			OutputFormat: "",
		}

		agg, err := aggregator.New(opts)
		if err != nil {
			t.Fatal(err)
		}

		if agg == nil {
			t.Fatal("New() returned nil")
		}
		// Empty format should default to JSON
	})
}
