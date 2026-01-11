package aggregator_test

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/valksor/go-assern/internal/aggregator"
)

func TestPrefixToolName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		server   string
		tool     string
		expected string
	}{
		{"github", "search", "github_search"},
		{"github", "search-repos", "github_search_repos"},
		{"my-server", "my-tool", "my_server_my_tool"},
		{"server", "tool_name", "server_tool_name"},
	}

	for _, tt := range tests {
		result := aggregator.PrefixToolName(tt.server, tt.tool)
		if result != tt.expected {
			t.Errorf("PrefixToolName(%q, %q) = %q, want %q",
				tt.server, tt.tool, result, tt.expected)
		}
	}
}

func TestParsePrefixedName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		prefixed       string
		expectedServer string
		expectedTool   string
	}{
		{"github_search", "github", "search"},
		{"github_search_repos", "github", "search_repos"},
		{"server_tool", "server", "tool"},
		{"invalid", "", ""},
	}

	for _, tt := range tests {
		server, tool := aggregator.ParsePrefixedName(tt.prefixed)
		if server != tt.expectedServer || tool != tt.expectedTool {
			t.Errorf("ParsePrefixedName(%q) = (%q, %q), want (%q, %q)",
				tt.prefixed, server, tool, tt.expectedServer, tt.expectedTool)
		}
	}
}

func TestToolRegistry_Register(t *testing.T) {
	t.Parallel()

	registry := aggregator.NewToolRegistry()

	tool := mcp.Tool{
		Name:        "search",
		Description: "Search for items",
	}

	registry.Register("github", tool, nil)

	if registry.Count() != 1 {
		t.Errorf("expected 1 tool, got %d", registry.Count())
	}

	entry, ok := registry.Get("github_search")
	if !ok {
		t.Fatal("tool not found")
	}

	if entry.ServerName != "github" {
		t.Errorf("expected server 'github', got '%s'", entry.ServerName)
	}

	if entry.Tool.Name != "search" {
		t.Errorf("expected tool name 'search', got '%s'", entry.Tool.Name)
	}
}

func TestToolRegistry_RegisterWithAllowed(t *testing.T) {
	t.Parallel()

	registry := aggregator.NewToolRegistry()

	allowed := []string{"search", "create"}

	tools := []mcp.Tool{
		{Name: "search"},
		{Name: "create"},
		{Name: "delete"}, // Not allowed
	}

	for _, tool := range tools {
		registry.Register("github", tool, allowed)
	}

	if registry.Count() != 2 {
		t.Errorf("expected 2 tools, got %d", registry.Count())
	}

	if _, ok := registry.Get("github_delete"); ok {
		t.Error("delete tool should not be registered")
	}
}

func TestToolRegistry_GetByServer(t *testing.T) {
	t.Parallel()

	registry := aggregator.NewToolRegistry()

	registry.Register("github", mcp.Tool{Name: "search"}, nil)
	registry.Register("github", mcp.Tool{Name: "create"}, nil)
	registry.Register("jira", mcp.Tool{Name: "get_ticket"}, nil)

	githubTools := registry.GetByServer("github")
	if len(githubTools) != 2 {
		t.Errorf("expected 2 github tools, got %d", len(githubTools))
	}

	jiraTools := registry.GetByServer("jira")
	if len(jiraTools) != 1 {
		t.Errorf("expected 1 jira tool, got %d", len(jiraTools))
	}
}

func TestToolRegistry_RemoveServer(t *testing.T) {
	t.Parallel()

	registry := aggregator.NewToolRegistry()

	registry.Register("github", mcp.Tool{Name: "search"}, nil)
	registry.Register("github", mcp.Tool{Name: "create"}, nil)
	registry.Register("jira", mcp.Tool{Name: "get_ticket"}, nil)

	if registry.Count() != 3 {
		t.Errorf("expected 3 tools, got %d", registry.Count())
	}

	registry.RemoveServer("github")

	if registry.Count() != 1 {
		t.Errorf("expected 1 tool after removal, got %d", registry.Count())
	}

	if _, ok := registry.Get("github_search"); ok {
		t.Error("github_search should not exist after removal")
	}
}

func TestToolRegistry_Clear(t *testing.T) {
	t.Parallel()

	registry := aggregator.NewToolRegistry()

	registry.Register("github", mcp.Tool{Name: "search"}, nil)
	registry.Register("jira", mcp.Tool{Name: "get_ticket"}, nil)

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("expected 0 tools after clear, got %d", registry.Count())
	}
}

func TestToolEntry_Summarize(t *testing.T) {
	t.Parallel()

	entry := &aggregator.ToolEntry{
		ServerName:   "github",
		PrefixedName: "github_search",
		Tool: mcp.Tool{
			Name:        "search",
			Description: "Search for repositories",
		},
	}

	summary := entry.Summarize()

	if summary.PrefixedName != "github_search" {
		t.Errorf("expected prefixed name 'github_search', got '%s'", summary.PrefixedName)
	}

	if summary.ServerName != "github" {
		t.Errorf("expected server name 'github', got '%s'", summary.ServerName)
	}

	if summary.Description != "Search for repositories" {
		t.Errorf("expected description 'Search for repositories', got '%s'", summary.Description)
	}
}

func TestPrefixToolName_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		server   string
		tool     string
		expected string
	}{
		{"empty server", "", "tool", "_tool"},
		{"empty tool", "server", "", "server_"},
		{"both empty", "", "", "_"},
		{"server with dashes", "my-server", "tool", "my_server_tool"},
		{"tool with dashes", "server", "my-tool", "server_my_tool"},
		{"both with dashes", "my-server", "my-tool", "my_server_my_tool"},
		{"server with underscores", "my_server", "tool", "my_server_tool"},
		{"tool with underscores", "server", "my_tool", "server_my_tool"},
		{"multiple dashes", "my-cool-server", "my-awesome-tool", "my_cool_server_my_awesome_tool"},
		{"special chars", "server-123", "tool_abc", "server_123_tool_abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := aggregator.PrefixToolName(tt.server, tt.tool)
			if result != tt.expected {
				t.Errorf("PrefixToolName(%q, %q) = %q, want %q",
					tt.server, tt.tool, result, tt.expected)
			}
		})
	}
}

func TestParsePrefixedName_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		prefixed       string
		expectedServer string
		expectedTool   string
	}{
		{"empty string", "", "", ""},
		{"no underscore", "invalid", "", ""},
		{"single underscore", "a_b", "a", "b"},
		{"multiple underscores", "a_b_c", "a", "b_c"},
		{"underscore at start", "_tool", "", "tool"},
		{"underscore at end", "server_", "server", ""},
		{"only underscore", "_", "", ""},
		{"single char before underscore", "a_b", "a", "b"},
		{"single char after underscore", "a_b", "a", "b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server, tool := aggregator.ParsePrefixedName(tt.prefixed)
			if server != tt.expectedServer || tool != tt.expectedTool {
				t.Errorf("ParsePrefixedName(%q) = (%q, %q), want (%q, %q)",
					tt.prefixed, server, tool, tt.expectedServer, tt.expectedTool)
			}
		})
	}
}

func TestToolRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	registry := aggregator.NewToolRegistry()

	// Test concurrent reads
	registry.Register("server1", mcp.Tool{Name: "tool1"}, nil)
	registry.Register("server2", mcp.Tool{Name: "tool2"}, nil)

	done := make(chan bool)

	// Start multiple goroutines reading
	for range 10 {
		go func() {
			defer func() { done <- true }()
			for range 100 {
				_ = registry.Count()
				_, _ = registry.Get("server1_tool1")
				_ = registry.All()
				_ = registry.GetByServer("server1")
				_ = registry.ServerCount()
			}
		}()
	}

	// Start goroutines writing
	for i := range 5 {
		go func(n int) {
			defer func() { done <- true }()
			for j := range 50 {
				registry.Register("server"+string(rune('0'+n)), mcp.Tool{Name: "tool" + string(rune('0'+j))}, nil)
			}
		}(i)
	}

	// Wait for all goroutines
	for range 15 {
		<-done
	}

	// Verify final state
	count := registry.Count()
	if count == 0 {
		t.Error("Expected non-zero count after concurrent operations")
	}
}

func TestToolRegistry_ServerCount(t *testing.T) {
	t.Parallel()

	registry := aggregator.NewToolRegistry()

	if registry.ServerCount() != 0 {
		t.Errorf("expected 0 servers, got %d", registry.ServerCount())
	}

	registry.Register("server1", mcp.Tool{Name: "tool1"}, nil)
	if registry.ServerCount() != 1 {
		t.Errorf("expected 1 server, got %d", registry.ServerCount())
	}

	registry.Register("server1", mcp.Tool{Name: "tool2"}, nil)
	if registry.ServerCount() != 1 {
		t.Errorf("expected 1 server (same server), got %d", registry.ServerCount())
	}

	registry.Register("server2", mcp.Tool{Name: "tool1"}, nil)
	if registry.ServerCount() != 2 {
		t.Errorf("expected 2 servers, got %d", registry.ServerCount())
	}

	registry.RemoveServer("server1")
	if registry.ServerCount() != 1 {
		t.Errorf("expected 1 server after removal, got %d", registry.ServerCount())
	}
}

func TestToolEntry_Summarize_EmptyDescription(t *testing.T) {
	t.Parallel()

	entry := &aggregator.ToolEntry{
		ServerName:   "github",
		PrefixedName: "github_search",
		Tool: mcp.Tool{
			Name:        "search",
			Description: "",
		},
	}

	summary := entry.Summarize()

	if summary.Description != "" {
		t.Errorf("expected empty description, got '%s'", summary.Description)
	}

	if summary.OriginalName != "search" {
		t.Errorf("expected original name 'search', got '%s'", summary.OriginalName)
	}
}

func TestToolRegistry_All(t *testing.T) {
	t.Parallel()

	registry := aggregator.NewToolRegistry()

	tools := []mcp.Tool{
		{Name: "tool1", Description: "First tool"},
		{Name: "tool2", Description: "Second tool"},
		{Name: "tool3", Description: "Third tool"},
	}

	for _, tool := range tools {
		registry.Register("server1", tool, nil)
	}

	all := registry.All()
	if len(all) != 3 {
		t.Errorf("expected 3 tools, got %d", len(all))
	}

	// Verify we can modify the returned slice without affecting the registry
	all[0] = nil
	if registry.Count() != 3 {
		t.Error("Modifying All() result affected the registry")
	}
}
