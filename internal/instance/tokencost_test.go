package instance

import (
	"encoding/json"
	"testing"
)

func TestEstimateListTokens(t *testing.T) {
	t.Parallel()

	tools := []ToolInfo{
		{Name: "github_search", Description: "Search repositories", InputSchema: json.RawMessage(`{"type":"object"}`)},
		{Name: "github_issue", Description: "Get an issue"},
		{Name: "linear_ticket", Description: "Get a ticket"},
		{Name: "noprefix", Description: "tool without a server prefix"},
	}

	byServer, total := estimateListTokens(tools)

	if total <= 0 {
		t.Fatalf("total tokens = %d, want > 0", total)
	}

	for _, server := range []string{"github", "linear", "(unprefixed)"} {
		if _, ok := byServer[server]; !ok {
			t.Errorf("expected bucket %q, got %v", server, byServer)
		}
	}

	// Two github tools must accumulate into a single bucket.
	if byServer["github"] <= byServer["linear"] {
		t.Errorf("github bucket (%d) should exceed single linear bucket (%d)", byServer["github"], byServer["linear"])
	}

	sum := 0
	for _, v := range byServer {
		sum += v
	}

	if sum != total {
		t.Errorf("sum of per-server (%d) != total (%d)", sum, total)
	}
}

func TestEstimateListTokensEmpty(t *testing.T) {
	t.Parallel()

	byServer, total := estimateListTokens(nil)

	if total != 0 {
		t.Errorf("total = %d, want 0 for empty input", total)
	}

	if len(byServer) != 0 {
		t.Errorf("byServer = %v, want empty", byServer)
	}
}
