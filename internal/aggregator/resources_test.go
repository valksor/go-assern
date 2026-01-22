package aggregator

import (
	"errors"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestResourceRegistry_Register(t *testing.T) {
	t.Parallel()

	registry := NewResourceRegistry()

	resource := mcp.NewResource("file:///test.txt", "Test File", mcp.WithResourceDescription("A test file"))
	registry.Register("server1", resource)

	if registry.Count() != 1 {
		t.Errorf("Count() = %d, want 1", registry.Count())
	}
}

func TestResourceRegistry_Get(t *testing.T) {
	t.Parallel()

	registry := NewResourceRegistry()

	resource := mcp.NewResource("file:///test.txt", "Test File")
	registry.Register("server1", resource)

	// Get existing entry
	entry, ok := registry.Get("assern://server1/file:///test.txt")
	if !ok {
		t.Error("Get() returned false for existing entry")
	}
	if entry.ServerName != "server1" {
		t.Errorf("entry.ServerName = %q, want 'server1'", entry.ServerName)
	}

	// Get non-existing entry
	_, ok = registry.Get("assern://nonexistent/file:///test.txt")
	if ok {
		t.Error("Get() returned true for non-existing entry")
	}
}

func TestResourceRegistry_GetByServer(t *testing.T) {
	t.Parallel()

	registry := NewResourceRegistry()

	resource1 := mcp.NewResource("file:///a.txt", "File A")
	resource2 := mcp.NewResource("file:///b.txt", "File B")
	registry.Register("server1", resource1)
	registry.Register("server1", resource2)

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

func TestResourceRegistry_All(t *testing.T) {
	t.Parallel()

	registry := NewResourceRegistry()

	resource1 := mcp.NewResource("file:///a.txt", "File A")
	resource2 := mcp.NewResource("file:///b.txt", "File B")
	registry.Register("server1", resource1)
	registry.Register("server2", resource2)

	all := registry.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d entries, want 2", len(all))
	}
}

func TestResourceRegistry_Clear(t *testing.T) {
	t.Parallel()

	registry := NewResourceRegistry()

	resource := mcp.NewResource("file:///test.txt", "Test")
	registry.Register("server1", resource)

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Count() after Clear() = %d, want 0", registry.Count())
	}
}

func TestResourceRegistry_RemoveServer(t *testing.T) {
	t.Parallel()

	registry := NewResourceRegistry()

	resource1 := mcp.NewResource("file:///a.txt", "File A")
	resource2 := mcp.NewResource("file:///b.txt", "File B")
	registry.Register("server1", resource1)
	registry.Register("server2", resource2)

	registry.RemoveServer("server1")

	if registry.Count() != 1 {
		t.Errorf("Count() after RemoveServer() = %d, want 1", registry.Count())
	}

	entries := registry.GetByServer("server1")
	if len(entries) != 0 {
		t.Errorf("GetByServer() after RemoveServer() returned %d entries, want 0", len(entries))
	}
}

func TestPrefixResourceURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		server   string
		uri      string
		expected string
	}{
		{"github", "file:///README.md", "assern://github/file:///README.md"},
		{"my-server", "http://example.com", "assern://my_server/http://example.com"},
		{"server_name", "custom://path", "assern://server_name/custom://path"},
	}

	for _, tt := range tests {
		t.Run(tt.server+"_"+tt.uri, func(t *testing.T) {
			t.Parallel()

			result := PrefixResourceURI(tt.server, tt.uri)
			if result != tt.expected {
				t.Errorf("PrefixResourceURI(%q, %q) = %q, want %q",
					tt.server, tt.uri, result, tt.expected)
			}
		})
	}
}

func TestParsePrefixedURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		prefixedURI    string
		expectedServer string
		expectedURI    string
		wantErr        bool
	}{
		{"assern://github/file:///README.md", "github", "file:///README.md", false},
		{"assern://server/http://example.com", "server", "http://example.com", false},
		{"file:///test.txt", "", "", true},   // Invalid - no assern:// prefix
		{"assern://server", "", "", true},    // Invalid - no slash after server
		{"http://example.com", "", "", true}, // Invalid - wrong prefix
		{"assern://s/uri/with/slashes", "s", "uri/with/slashes", false},
	}

	for _, tt := range tests {
		t.Run(tt.prefixedURI, func(t *testing.T) {
			t.Parallel()

			server, uri, err := ParsePrefixedURI(tt.prefixedURI)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePrefixedURI(%q) expected error, got nil", tt.prefixedURI)
				}
				if !errors.Is(err, ErrInvalidPrefixedURI) {
					t.Errorf("ParsePrefixedURI(%q) error = %v, want ErrInvalidPrefixedURI", tt.prefixedURI, err)
				}
			} else {
				if err != nil {
					t.Errorf("ParsePrefixedURI(%q) unexpected error: %v", tt.prefixedURI, err)
				}
				if server != tt.expectedServer {
					t.Errorf("ParsePrefixedURI(%q) server = %q, want %q",
						tt.prefixedURI, server, tt.expectedServer)
				}
				if uri != tt.expectedURI {
					t.Errorf("ParsePrefixedURI(%q) uri = %q, want %q",
						tt.prefixedURI, uri, tt.expectedURI)
				}
			}
		})
	}
}

func TestResourceRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	registry := NewResourceRegistry()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := range 10 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resource := mcp.NewResource("file:///test"+string(rune('0'+idx))+".txt", "Test")
			registry.Register("server"+string(rune('0'+idx)), resource)
		}(i)
	}

	// Concurrent reads
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = registry.Count()
			_ = registry.All()
			_, _ = registry.Get("assern://server0/file:///test0.txt")
			_ = registry.GetByServer("server0")
		}()
	}

	wg.Wait()

	// No race conditions should occur
}
