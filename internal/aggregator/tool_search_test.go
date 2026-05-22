package aggregator_test

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/valksor/go-assern/internal/aggregator"
)

// newSearchRegistry builds a registry populated with a small, known catalog.
func newSearchRegistry(t *testing.T) *aggregator.ToolRegistry {
	t.Helper()

	reg := aggregator.NewToolRegistry()
	reg.Register("github", mcp.NewTool("search_repos", mcp.WithDescription("Search repositories by query")), nil)
	reg.Register("github", mcp.NewTool("create_issue", mcp.WithDescription("Open a new issue in a repository")), nil)
	reg.Register("linear", mcp.NewTool("search", mcp.WithDescription("Search tickets and projects")), nil)
	reg.Register("filesystem", mcp.NewTool("read_file", mcp.WithDescription("Read a file from disk")), nil)

	return reg
}

func prefixedNames(entries []*aggregator.ToolEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.PrefixedName
	}

	return names
}

func TestToolRegistrySearchRanksNameMatchesFirst(t *testing.T) {
	t.Parallel()

	reg := newSearchRegistry(t)

	results := reg.Search("search", 0)

	if len(results) < 2 {
		t.Fatalf("expected at least 2 matches for 'search', got %v", prefixedNames(results))
	}

	// Tools with "search" as a name word must rank above description-only matches.
	top := prefixedNames(results)[:2]
	for _, name := range top {
		if name != "github_search_repos" && name != "linear_search" {
			t.Errorf("unexpected top result %q; top two = %v", name, top)
		}
	}

	// read_file does not mention search anywhere and must be excluded.
	for _, name := range prefixedNames(results) {
		if name == "filesystem_read_file" {
			t.Errorf("filesystem_read_file should not match 'search': %v", prefixedNames(results))
		}
	}
}

func TestToolRegistrySearchLimit(t *testing.T) {
	t.Parallel()

	reg := newSearchRegistry(t)

	results := reg.Search("search repositories", 1)
	if len(results) != 1 {
		t.Fatalf("limit 1 returned %d results: %v", len(results), prefixedNames(results))
	}

	// "search repositories" should rank github_search_repos first (matches both terms in the name).
	if results[0].PrefixedName != "github_search_repos" {
		t.Errorf("top result = %q, want github_search_repos", results[0].PrefixedName)
	}
}

func TestToolRegistrySearchEmptyQueryBrowses(t *testing.T) {
	t.Parallel()

	reg := newSearchRegistry(t)

	results := reg.Search("", 0)
	if len(results) != 4 {
		t.Fatalf("empty query returned %d results, want all 4", len(results))
	}

	// Empty query returns name-sorted entries.
	names := prefixedNames(results)
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("results not sorted by name: %v", names)

			break
		}
	}

	// Empty query still honours the limit.
	if got := reg.Search("", 2); len(got) != 2 {
		t.Errorf("empty query with limit 2 returned %d results", len(got))
	}
}

func TestToolRegistrySearchNoMatch(t *testing.T) {
	t.Parallel()

	reg := newSearchRegistry(t)

	if results := reg.Search("nonexistentxyz", 0); len(results) != 0 {
		t.Errorf("expected no matches, got %v", prefixedNames(results))
	}
}

func TestToolRegistrySearchServerNameMatch(t *testing.T) {
	t.Parallel()

	reg := newSearchRegistry(t)

	results := reg.Search("filesystem", 0)
	if len(results) != 1 || results[0].PrefixedName != "filesystem_read_file" {
		t.Errorf("search by server name returned %v, want [filesystem_read_file]", prefixedNames(results))
	}
}
