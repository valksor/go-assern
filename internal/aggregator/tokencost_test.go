package aggregator_test

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/valksor/go-assern/internal/aggregator"
)

func TestEstimateToolTokens(t *testing.T) {
	t.Parallel()

	nameOnly := mcp.NewTool("ping")
	withDesc := mcp.NewTool("ping", mcp.WithDescription("Check whether the server is alive"))
	withSchema := mcp.NewTool(
		"search",
		mcp.WithDescription("Search repositories by a free-text query string"),
		mcp.WithString("query", mcp.Required(), mcp.Description("the search query")),
		mcp.WithNumber("limit", mcp.Description("maximum number of results to return")),
	)

	nameOnlyTok := aggregator.EstimateToolTokens(nameOnly)
	withDescTok := aggregator.EstimateToolTokens(withDesc)
	withSchemaTok := aggregator.EstimateToolTokens(withSchema)

	if nameOnlyTok <= 0 {
		t.Errorf("EstimateToolTokens(name-only) = %d, want > 0", nameOnlyTok)
	}

	// More content must never cost fewer tokens.
	if withDescTok < nameOnlyTok {
		t.Errorf("description tool (%d) cheaper than name-only (%d)", withDescTok, nameOnlyTok)
	}

	if withSchemaTok <= withDescTok {
		t.Errorf("schema tool (%d) not more expensive than description-only (%d)", withSchemaTok, withDescTok)
	}
}

func TestEstimateToolTokensRawConsistency(t *testing.T) {
	t.Parallel()

	tool := mcp.NewTool(
		"search",
		mcp.WithDescription("Search repositories"),
		mcp.WithString("query", mcp.Required()),
	)

	schema, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatalf("marshal input schema: %v", err)
	}

	fromTool := aggregator.EstimateToolTokens(tool)
	fromRaw := aggregator.EstimateRawToolTokens(tool.Name, tool.Description, schema)

	if fromTool != fromRaw {
		t.Errorf("EstimateToolTokens=%d, EstimateRawToolTokens=%d; want equal", fromTool, fromRaw)
	}
}

func TestEstimateRawToolTokensEmpty(t *testing.T) {
	t.Parallel()

	// An empty definition still serializes to a small JSON object, so the
	// estimate is small but non-negative.
	got := aggregator.EstimateRawToolTokens("", "", nil)
	if got < 0 {
		t.Errorf("EstimateRawToolTokens(empty) = %d, want >= 0", got)
	}
}

func TestEstimateCatalogTokens(t *testing.T) {
	t.Parallel()

	entries := []*aggregator.ToolEntry{
		{
			ServerName:   "github",
			Tool:         mcp.NewTool("search", mcp.WithDescription("Search repos")),
			PrefixedName: "github_search",
		},
		{
			ServerName:   "github",
			Tool:         mcp.NewTool("issue", mcp.WithDescription("Get an issue")),
			PrefixedName: "github_issue",
		},
		{
			ServerName:   "linear",
			Tool:         mcp.NewTool("ticket", mcp.WithDescription("Get a ticket")),
			PrefixedName: "linear_ticket",
		},
		nil, // must be skipped, not panic
	}

	perServer, total := aggregator.EstimateCatalogTokens(entries)

	if len(perServer) != 2 {
		t.Fatalf("perServer has %d servers, want 2: %v", len(perServer), perServer)
	}

	if perServer["github"] <= 0 || perServer["linear"] <= 0 {
		t.Errorf("per-server token costs not positive: %v", perServer)
	}

	if want := perServer["github"] + perServer["linear"]; total != want {
		t.Errorf("total = %d, want sum of per-server = %d", total, want)
	}
}

func TestExposedTool(t *testing.T) {
	t.Parallel()

	entry := &aggregator.ToolEntry{
		ServerName:   "github",
		Tool:         mcp.NewTool("search", mcp.WithDescription("Search repos")),
		PrefixedName: "github_search",
	}

	exposed := entry.ExposedTool()

	if exposed.Name != "github_search" {
		t.Errorf("exposed name = %q, want %q", exposed.Name, "github_search")
	}

	if exposed.Description != entry.Tool.Description {
		t.Errorf("exposed description = %q, want %q", exposed.Description, entry.Tool.Description)
	}
}
