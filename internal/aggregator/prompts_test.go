package aggregator

import (
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestPromptRegistry_Register(t *testing.T) {
	t.Parallel()

	registry := NewPromptRegistry()

	prompt := mcp.Prompt{Name: "test-prompt", Description: "A test prompt"}
	registry.Register("server1", prompt)

	if registry.Count() != 1 {
		t.Errorf("Count() = %d, want 1", registry.Count())
	}
}

func TestPromptRegistry_Get(t *testing.T) {
	t.Parallel()

	registry := NewPromptRegistry()

	prompt := mcp.Prompt{Name: "test-prompt", Description: "A test prompt"}
	registry.Register("server1", prompt)

	// Get existing entry
	entry, ok := registry.Get("server1_test_prompt")
	if !ok {
		t.Error("Get() returned false for existing entry")
	}
	if entry.ServerName != "server1" {
		t.Errorf("entry.ServerName = %q, want 'server1'", entry.ServerName)
	}

	// Get non-existing entry
	_, ok = registry.Get("nonexistent_prompt")
	if ok {
		t.Error("Get() returned true for non-existing entry")
	}
}

func TestPromptRegistry_GetByServer(t *testing.T) {
	t.Parallel()

	registry := NewPromptRegistry()

	prompt1 := mcp.Prompt{Name: "prompt1", Description: "Prompt 1"}
	prompt2 := mcp.Prompt{Name: "prompt2", Description: "Prompt 2"}
	registry.Register("server1", prompt1)
	registry.Register("server1", prompt2)

	entries := registry.GetByServer("server1")
	if len(entries) != 2 {
		t.Errorf("GetByServer() returned %d entries, want 2", len(entries))
	}

	// Non-existent server
	entries = registry.GetByServer("nonexistent")
	if len(entries) != 0 {
		t.Errorf("GetByServer() returned %d entries for non-existent server, want 0", len(entries))
	}
}

func TestPromptRegistry_All(t *testing.T) {
	t.Parallel()

	registry := NewPromptRegistry()

	prompt1 := mcp.Prompt{Name: "prompt1", Description: "Prompt 1"}
	prompt2 := mcp.Prompt{Name: "prompt2", Description: "Prompt 2"}
	registry.Register("server1", prompt1)
	registry.Register("server2", prompt2)

	all := registry.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d entries, want 2", len(all))
	}
}

func TestPromptRegistry_Clear(t *testing.T) {
	t.Parallel()

	registry := NewPromptRegistry()

	prompt := mcp.Prompt{Name: "test", Description: "Test"}
	registry.Register("server1", prompt)

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Count() after Clear() = %d, want 0", registry.Count())
	}
}

func TestPromptRegistry_RemoveServer(t *testing.T) {
	t.Parallel()

	registry := NewPromptRegistry()

	prompt1 := mcp.Prompt{Name: "prompt1", Description: "Prompt 1"}
	prompt2 := mcp.Prompt{Name: "prompt2", Description: "Prompt 2"}
	registry.Register("server1", prompt1)
	registry.Register("server2", prompt2)

	registry.RemoveServer("server1")

	if registry.Count() != 1 {
		t.Errorf("Count() after RemoveServer() = %d, want 1", registry.Count())
	}

	entries := registry.GetByServer("server1")
	if len(entries) != 0 {
		t.Errorf("GetByServer() after RemoveServer() returned %d entries, want 0", len(entries))
	}
}

func TestPrefixPromptName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		server     string
		promptName string
		expected   string
	}{
		{"github", "create-issue", "github_create_issue"},
		{"my-server", "my-prompt", "my_server_my_prompt"},
		{"server_name", "prompt_name", "server_name_prompt_name"},
		{"server", "prompt", "server_prompt"},
	}

	for _, tt := range tests {
		t.Run(tt.server+"_"+tt.promptName, func(t *testing.T) {
			t.Parallel()

			result := PrefixPromptName(tt.server, tt.promptName)
			if result != tt.expected {
				t.Errorf("PrefixPromptName(%q, %q) = %q, want %q",
					tt.server, tt.promptName, result, tt.expected)
			}
		})
	}
}

func TestParsePrefixedPromptName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		prefixedName   string
		expectedServer string
		expectedPrompt string
	}{
		{"github_create_issue", "github", "create_issue"},
		{"server_prompt_name", "server", "prompt_name"},
		{"nounderscore", "", ""},  // Invalid - no underscore
		{"_prompt", "", "prompt"}, // Edge case - empty server
		{"server_", "server", ""}, // Edge case - empty prompt
		{"s_p", "s", "p"},         // Minimal valid
		{"a_b_c", "a", "b_c"},     // Multiple underscores
	}

	for _, tt := range tests {
		t.Run(tt.prefixedName, func(t *testing.T) {
			t.Parallel()

			server, prompt := ParsePrefixedPromptName(tt.prefixedName)
			if server != tt.expectedServer {
				t.Errorf("ParsePrefixedPromptName(%q) server = %q, want %q",
					tt.prefixedName, server, tt.expectedServer)
			}
			if prompt != tt.expectedPrompt {
				t.Errorf("ParsePrefixedPromptName(%q) prompt = %q, want %q",
					tt.prefixedName, prompt, tt.expectedPrompt)
			}
		})
	}
}

func TestPromptRegistry_WithArguments(t *testing.T) {
	t.Parallel()

	registry := NewPromptRegistry()

	prompt := mcp.Prompt{
		Name:        "generate-code",
		Description: "Generate code from template",
		Arguments: []mcp.PromptArgument{
			{Name: "language", Description: "Programming language", Required: true},
			{Name: "framework", Description: "Framework to use", Required: false},
		},
	}
	registry.Register("templates", prompt)

	entry, ok := registry.Get("templates_generate_code")
	if !ok {
		t.Fatal("Get() returned false")
	}

	if len(entry.Prompt.Arguments) != 2 {
		t.Errorf("Arguments count = %d, want 2", len(entry.Prompt.Arguments))
	}

	if entry.Prompt.Arguments[0].Name != "language" {
		t.Errorf("First argument name = %q, want 'language'", entry.Prompt.Arguments[0].Name)
	}
}

func TestPromptRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	registry := NewPromptRegistry()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := range 10 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			prompt := mcp.Prompt{
				Name:        "prompt" + string(rune('0'+idx)),
				Description: "Test",
			}
			registry.Register("server"+string(rune('0'+idx)), prompt)
		}(i)
	}

	// Concurrent reads
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = registry.Count()
			_ = registry.All()
			_, _ = registry.Get("server0_prompt0")
			_ = registry.GetByServer("server0")
		}()
	}

	wg.Wait()

	// No race conditions should occur
}
