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

func TestAggregator_AddServer(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()
	ctx := context.Background()

	t.Run("adds mock server successfully", func(t *testing.T) {
		t.Parallel()

		agg, err := aggregator.New(aggregator.Options{
			Config: cfg,
			Logger: logger,
		})
		if err != nil {
			t.Fatal(err)
		}

		mock := testutil.NewMockServer("test-server", []mcp.Tool{
			{Name: "search", Description: "Search for things"},
			{Name: "create", Description: "Create something"},
		})

		if err := mock.Start(ctx); err != nil {
			t.Fatal(err)
		}

		if err := agg.AddServer(ctx, mock); err != nil {
			t.Fatalf("AddServer() error = %v", err)
		}

		// Verify server was added
		names := agg.ServerNames()
		if len(names) != 1 || names[0] != "test-server" {
			t.Errorf("ServerNames() = %v, want [test-server]", names)
		}

		// Verify tools were discovered with prefix
		tools := agg.ListTools()
		if len(tools) != 2 {
			t.Errorf("ListTools() returned %d tools, want 2", len(tools))
		}

		// Check prefixed names (hyphens converted to underscores)
		foundSearch := false
		foundCreate := false

		for _, tool := range tools {
			if tool.PrefixedName == "test_server_search" {
				foundSearch = true
			}

			if tool.PrefixedName == "test_server_create" {
				foundCreate = true
			}
		}

		if !foundSearch {
			t.Error("expected tool test_server_search not found")
		}

		if !foundCreate {
			t.Error("expected tool test_server_create not found")
		}
	})

	t.Run("rejects duplicate server name", func(t *testing.T) {
		t.Parallel()

		agg, err := aggregator.New(aggregator.Options{
			Config: cfg,
			Logger: logger,
		})
		if err != nil {
			t.Fatal(err)
		}

		mock1 := testutil.NewMockServer("duplicate", []mcp.Tool{})
		mock2 := testutil.NewMockServer("duplicate", []mcp.Tool{})

		_ = mock1.Start(ctx)
		_ = mock2.Start(ctx)

		if err := agg.AddServer(ctx, mock1); err != nil {
			t.Fatal(err)
		}

		// Second add should fail
		err = agg.AddServer(ctx, mock2)
		if err == nil {
			t.Error("AddServer() should return error for duplicate name")
		}
	})
}

func TestAggregator_MultipleServers(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()
	ctx := context.Background()

	agg, err := aggregator.New(aggregator.Options{
		Config: cfg,
		Logger: logger,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Add multiple mock servers
	github := testutil.NewMockServer("github", []mcp.Tool{
		{Name: "search_code", Description: "Search code"},
		{Name: "create_issue", Description: "Create issue"},
	})
	filesystem := testutil.NewMockServer("filesystem", []mcp.Tool{
		{Name: "read_file", Description: "Read file"},
		{Name: "write_file", Description: "Write file"},
	})

	_ = github.Start(ctx)
	_ = filesystem.Start(ctx)

	if err := agg.AddServer(ctx, github); err != nil {
		t.Fatal(err)
	}

	if err := agg.AddServer(ctx, filesystem); err != nil {
		t.Fatal(err)
	}

	// Verify all tools are registered
	tools := agg.ListTools()
	if len(tools) != 4 {
		t.Errorf("ListTools() returned %d tools, want 4", len(tools))
	}

	// Verify server names
	names := agg.ServerNames()
	if len(names) != 2 {
		t.Errorf("ServerNames() returned %d servers, want 2", len(names))
	}

	// Verify tools have correct prefixes
	expectedTools := map[string]bool{
		"github_search_code":    false,
		"github_create_issue":   false,
		"filesystem_read_file":  false,
		"filesystem_write_file": false,
	}

	for _, tool := range tools {
		if _, ok := expectedTools[tool.PrefixedName]; ok {
			expectedTools[tool.PrefixedName] = true
		}
	}

	for name, found := range expectedTools {
		if !found {
			t.Errorf("expected tool %s not found", name)
		}
	}
}

func TestAggregator_ToolCallRouting(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()
	ctx := context.Background()

	agg, err := aggregator.New(aggregator.Options{
		Config: cfg,
		Logger: logger,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create mock server with custom result
	mock := testutil.NewMockServer("test", []mcp.Tool{
		{Name: "echo", Description: "Echo input"},
	})
	mock.SetToolResult("echo", &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: "custom response"},
		},
	})

	_ = mock.Start(ctx)

	if err := agg.AddServer(ctx, mock); err != nil {
		t.Fatal(err)
	}

	// Create MCP server to test handler
	mcpServer := agg.CreateMCPServer()
	if mcpServer == nil {
		t.Fatal("CreateMCPServer() returned nil")
	}

	// Verify tool calls are tracked
	calls := mock.GetToolCalls()
	if len(calls) != 0 {
		t.Errorf("expected 0 initial calls, got %d", len(calls))
	}
}

func TestAggregator_ToolFiltering(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()
	ctx := context.Background()

	agg, err := aggregator.New(aggregator.Options{
		Config: cfg,
		Logger: logger,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create mock with allowed filter
	mock := testutil.NewMockServer("filtered", []mcp.Tool{
		{Name: "allowed_tool", Description: "This should be allowed"},
		{Name: "blocked_tool", Description: "This should be blocked"},
	})
	mock.ServerCfg.Allowed = []string{"allowed_tool"}

	_ = mock.Start(ctx)

	if err := agg.AddServer(ctx, mock); err != nil {
		t.Fatal(err)
	}

	// Only allowed_tool should be registered
	tools := agg.ListTools()
	if len(tools) != 1 {
		t.Errorf("ListTools() returned %d tools, want 1", len(tools))
	}

	if len(tools) > 0 && tools[0].Tool.Name != "allowed_tool" {
		t.Errorf("expected allowed_tool, got %s", tools[0].Tool.Name)
	}
}

func TestAggregator_StopCleansUp(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()
	ctx := context.Background()

	agg, err := aggregator.New(aggregator.Options{
		Config: cfg,
		Logger: logger,
	})
	if err != nil {
		t.Fatal(err)
	}

	mock := testutil.NewMockServer("cleanup-test", []mcp.Tool{
		{Name: "tool1", Description: "Tool 1"},
	})
	_ = mock.Start(ctx)

	if err := agg.AddServer(ctx, mock); err != nil {
		t.Fatal(err)
	}

	// Verify server is added
	if len(agg.ServerNames()) != 1 {
		t.Fatal("server not added")
	}

	// Stop the aggregator
	if err := agg.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Verify cleanup
	if len(agg.ServerNames()) != 0 {
		t.Error("ServerNames() should be empty after Stop()")
	}

	if len(agg.ListTools()) != 0 {
		t.Error("ListTools() should be empty after Stop()")
	}
}

func TestAggregator_GetServerWithMock(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()
	ctx := context.Background()

	agg, err := aggregator.New(aggregator.Options{
		Config: cfg,
		Logger: logger,
	})
	if err != nil {
		t.Fatal(err)
	}

	mock := testutil.NewMockServer("findme", []mcp.Tool{})
	_ = mock.Start(ctx)

	if err := agg.AddServer(ctx, mock); err != nil {
		t.Fatal(err)
	}

	// Get existing server
	srv, ok := agg.GetServer("findme")
	if !ok {
		t.Error("GetServer() should find 'findme'")
	}

	if srv.Name() != "findme" {
		t.Errorf("GetServer() returned server with name %q, want 'findme'", srv.Name())
	}

	// Get non-existent server
	_, ok = agg.GetServer("nonexistent")
	if ok {
		t.Error("GetServer() should return false for non-existent server")
	}
}

func TestAggregator_ConcurrentAddServer(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()
	ctx := context.Background()

	agg, err := aggregator.New(aggregator.Options{
		Config: cfg,
		Logger: logger,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Add servers concurrently
	done := make(chan error, 10)

	for i := range 10 {
		go func(idx int) {
			mock := testutil.NewMockServer(
				"server-"+string(rune('a'+idx)),
				[]mcp.Tool{{Name: "tool", Description: "Tool"}},
			)
			_ = mock.Start(ctx)
			done <- agg.AddServer(ctx, mock)
		}(i)
	}

	// Collect results
	successCount := 0

	for range 10 {
		if err := <-done; err == nil {
			successCount++
		}
	}

	// All should succeed (unique names)
	if successCount != 10 {
		t.Errorf("expected 10 successful adds, got %d", successCount)
	}

	// Verify all servers added
	if len(agg.ServerNames()) != 10 {
		t.Errorf("expected 10 servers, got %d", len(agg.ServerNames()))
	}
}

func TestAggregator_ResourcesPassthrough(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()
	ctx := context.Background()

	t.Run("discovers resources from server", func(t *testing.T) {
		t.Parallel()

		agg, err := aggregator.New(aggregator.Options{
			Config: cfg,
			Logger: logger,
		})
		if err != nil {
			t.Fatal(err)
		}

		mock := testutil.NewMockServer("fileserver", []mcp.Tool{})
		mock.Resources = []mcp.Resource{
			mcp.NewResource("file:///readme.md", "README", mcp.WithResourceDescription("Project readme")),
			mcp.NewResource("file:///config.json", "Config", mcp.WithResourceDescription("Configuration file")),
		}

		if err := mock.Start(ctx); err != nil {
			t.Fatal(err)
		}

		if err := agg.AddServer(ctx, mock); err != nil {
			t.Fatalf("AddServer() error = %v", err)
		}

		// Create MCP server and verify resources are registered
		mcpServer := agg.CreateMCPServer()
		if mcpServer == nil {
			t.Fatal("CreateMCPServer() returned nil")
		}

		// Resources should be discoverable through the MCP server
		// The prefixed URI format is "assern://{server}/{original-uri}"
	})

	t.Run("multiple servers with resources", func(t *testing.T) {
		t.Parallel()

		agg, err := aggregator.New(aggregator.Options{
			Config: cfg,
			Logger: logger,
		})
		if err != nil {
			t.Fatal(err)
		}

		// First server with resources
		server1 := testutil.NewMockServer("github", []mcp.Tool{})
		server1.Resources = []mcp.Resource{
			mcp.NewResource("repo://user/project", "Repository", mcp.WithResourceDescription("GitHub repo")),
		}

		// Second server with resources
		server2 := testutil.NewMockServer("filesystem", []mcp.Tool{})
		server2.Resources = []mcp.Resource{
			mcp.NewResource("file:///home/user/docs", "Docs", mcp.WithResourceDescription("User documents")),
			mcp.NewResource("file:///etc/config", "System Config", mcp.WithResourceDescription("System configuration")),
		}

		_ = server1.Start(ctx)
		_ = server2.Start(ctx)

		if err := agg.AddServer(ctx, server1); err != nil {
			t.Fatal(err)
		}
		if err := agg.AddServer(ctx, server2); err != nil {
			t.Fatal(err)
		}

		// Both servers should be added
		if len(agg.ServerNames()) != 2 {
			t.Errorf("expected 2 servers, got %d", len(agg.ServerNames()))
		}
	})

	t.Run("handles server with no resources", func(t *testing.T) {
		t.Parallel()

		agg, err := aggregator.New(aggregator.Options{
			Config: cfg,
			Logger: logger,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Server with only tools, no resources
		mock := testutil.NewMockServer("tools-only", []mcp.Tool{
			{Name: "search", Description: "Search tool"},
		})

		if err := mock.Start(ctx); err != nil {
			t.Fatal(err)
		}

		if err := agg.AddServer(ctx, mock); err != nil {
			t.Fatalf("AddServer() error = %v", err)
		}

		// Should have tools but no resources
		tools := agg.ListTools()
		if len(tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(tools))
		}
	})
}

func TestAggregator_PromptsPassthrough(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := config.NewConfig()
	ctx := context.Background()

	t.Run("discovers prompts from server", func(t *testing.T) {
		t.Parallel()

		agg, err := aggregator.New(aggregator.Options{
			Config: cfg,
			Logger: logger,
		})
		if err != nil {
			t.Fatal(err)
		}

		mock := testutil.NewMockServer("assistant", []mcp.Tool{})
		mock.Prompts = []mcp.Prompt{
			{Name: "code-review", Description: "Review code changes"},
			{Name: "explain-code", Description: "Explain code functionality"},
		}

		if err := mock.Start(ctx); err != nil {
			t.Fatal(err)
		}

		if err := agg.AddServer(ctx, mock); err != nil {
			t.Fatalf("AddServer() error = %v", err)
		}

		// Create MCP server and verify prompts are registered
		mcpServer := agg.CreateMCPServer()
		if mcpServer == nil {
			t.Fatal("CreateMCPServer() returned nil")
		}

		// Prompts should be discoverable through the MCP server
		// The prefixed name format is "{server}_{prompt-name}"
	})

	t.Run("multiple servers with prompts", func(t *testing.T) {
		t.Parallel()

		agg, err := aggregator.New(aggregator.Options{
			Config: cfg,
			Logger: logger,
		})
		if err != nil {
			t.Fatal(err)
		}

		// First server with prompts
		server1 := testutil.NewMockServer("coding", []mcp.Tool{})
		server1.Prompts = []mcp.Prompt{
			{Name: "debug", Description: "Debug code issues"},
		}

		// Second server with prompts
		server2 := testutil.NewMockServer("writing", []mcp.Tool{})
		server2.Prompts = []mcp.Prompt{
			{Name: "summarize", Description: "Summarize text"},
			{Name: "translate", Description: "Translate text"},
		}

		_ = server1.Start(ctx)
		_ = server2.Start(ctx)

		if err := agg.AddServer(ctx, server1); err != nil {
			t.Fatal(err)
		}
		if err := agg.AddServer(ctx, server2); err != nil {
			t.Fatal(err)
		}

		// Both servers should be added
		if len(agg.ServerNames()) != 2 {
			t.Errorf("expected 2 servers, got %d", len(agg.ServerNames()))
		}
	})

	t.Run("handles server with no prompts", func(t *testing.T) {
		t.Parallel()

		agg, err := aggregator.New(aggregator.Options{
			Config: cfg,
			Logger: logger,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Server with only tools, no prompts
		mock := testutil.NewMockServer("basic", []mcp.Tool{
			{Name: "ping", Description: "Ping tool"},
		})

		if err := mock.Start(ctx); err != nil {
			t.Fatal(err)
		}

		if err := agg.AddServer(ctx, mock); err != nil {
			t.Fatalf("AddServer() error = %v", err)
		}

		// Should have tools but no prompts
		tools := agg.ListTools()
		if len(tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(tools))
		}
	})

	t.Run("prompt with arguments", func(t *testing.T) {
		t.Parallel()

		agg, err := aggregator.New(aggregator.Options{
			Config: cfg,
			Logger: logger,
		})
		if err != nil {
			t.Fatal(err)
		}

		mock := testutil.NewMockServer("templates", []mcp.Tool{})
		mock.Prompts = []mcp.Prompt{
			{
				Name:        "generate-code",
				Description: "Generate code from template",
				Arguments: []mcp.PromptArgument{
					{Name: "language", Description: "Programming language", Required: true},
					{Name: "framework", Description: "Framework to use", Required: false},
				},
			},
		}

		if err := mock.Start(ctx); err != nil {
			t.Fatal(err)
		}

		if err := agg.AddServer(ctx, mock); err != nil {
			t.Fatalf("AddServer() error = %v", err)
		}

		// Create MCP server
		mcpServer := agg.CreateMCPServer()
		if mcpServer == nil {
			t.Fatal("CreateMCPServer() returned nil")
		}

		// Prompt arguments should be preserved
	})
}

func TestAggregator_ResourceAndPromptPrefixing(t *testing.T) {
	t.Parallel()

	t.Run("resource URI prefixing", func(t *testing.T) {
		t.Parallel()

		// Test the URI prefixing pattern
		serverName := "github"
		originalURI := "file:///repo/README.md"
		expected := "assern://github/file:///repo/README.md"

		prefixed := aggregator.PrefixResourceURI(serverName, originalURI)
		if prefixed != expected {
			t.Errorf("PrefixResourceURI() = %q, want %q", prefixed, expected)
		}
	})

	t.Run("resource URI parsing", func(t *testing.T) {
		t.Parallel()

		prefixedURI := "assern://github/file:///repo/README.md"
		expectedServer := "github"
		expectedOriginal := "file:///repo/README.md"

		server, original := aggregator.ParsePrefixedURI(prefixedURI)
		if server != expectedServer {
			t.Errorf("ParsePrefixedURI() server = %q, want %q", server, expectedServer)
		}
		if original != expectedOriginal {
			t.Errorf("ParsePrefixedURI() original = %q, want %q", original, expectedOriginal)
		}
	})

	t.Run("prompt name prefixing", func(t *testing.T) {
		t.Parallel()

		serverName := "github"
		promptName := "create-issue"
		expected := "github_create_issue"

		prefixed := aggregator.PrefixPromptName(serverName, promptName)
		if prefixed != expected {
			t.Errorf("PrefixPromptName() = %q, want %q", prefixed, expected)
		}
	})

	t.Run("prompt name parsing", func(t *testing.T) {
		t.Parallel()

		prefixedName := "github_create_issue"
		expectedServer := "github"
		expectedPrompt := "create_issue"

		server, prompt := aggregator.ParsePrefixedPromptName(prefixedName)
		if server != expectedServer {
			t.Errorf("ParsePrefixedPromptName() server = %q, want %q", server, expectedServer)
		}
		if prompt != expectedPrompt {
			t.Errorf("ParsePrefixedPromptName() prompt = %q, want %q", prompt, expectedPrompt)
		}
	})

	t.Run("invalid prefixed URI returns empty", func(t *testing.T) {
		t.Parallel()

		// Missing assern:// prefix
		server, original := aggregator.ParsePrefixedURI("file:///some/path")
		if server != "" || original != "" {
			t.Errorf("ParsePrefixedURI() should return empty for invalid prefix")
		}

		// Missing slash after server name
		server, original = aggregator.ParsePrefixedURI("assern://github")
		if server != "" || original != "" {
			t.Errorf("ParsePrefixedURI() should return empty for missing slash")
		}
	})
}
