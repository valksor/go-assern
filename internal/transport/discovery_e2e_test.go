package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/valksor/go-assern/internal/aggregator"
	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/testutil"
)

// rpcResp is a partial view of a JSON-RPC response sufficient to inspect
// tools/list and tools/call results emitted by the serve loop.
type rpcResp struct {
	ID     *int `json:"id"`
	Result struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"result"`
}

// TestDiscoveryEndToEndOverStdio drives the real stdio serve loop against a
// discovery-enabled aggregator: backend tools are hidden until searched,
// loaded per session, then callable and routed to the backend.
func TestDiscoveryEndToEndOverStdio(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := &config.Config{
		Servers:  map[string]*config.ServerConfig{},
		Settings: &config.Settings{Discovery: &config.DiscoveryConfig{Enabled: true}},
	}

	agg, err := aggregator.New(aggregator.Options{
		Config: cfg,
		Logger: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatalf("aggregator.New: %v", err)
	}

	mock := testutil.NewMockServer("github", []mcp.Tool{
		mcp.NewTool("search_repos", mcp.WithDescription("Search repositories")),
		mcp.NewTool("create_issue", mcp.WithDescription("Open an issue")),
	})
	if startErr := mock.Start(ctx); startErr != nil {
		t.Fatalf("mock.Start: %v", startErr)
	}

	if addErr := agg.AddServer(ctx, mock); addErr != nil {
		t.Fatalf("AddServer: %v", addErr)
	}

	srv := agg.CreateMCPServer()

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"assern_load","arguments":{"names":["github_search_repos"]}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"github_search_repos","arguments":{}}}`,
	}, "\n") + "\n"

	var buf bytes.Buffer

	logger := slog.New(slog.DiscardHandler)

	if loopErr := runSessionLoop(ctx, srv, newStdioSession(), strings.NewReader(input), &buf, logger); loopErr != nil {
		t.Fatalf("runSessionLoop: %v", loopErr)
	}

	byID := responsesByID(t, buf.String())

	// First tools/list (id 2): only meta-tools, backend tools hidden.
	firstList := toolNames(byID[2])
	if !slices.Contains(firstList, aggregator.ToolSearchName) {
		t.Errorf("initial tools/list missing %s: %v", aggregator.ToolSearchName, firstList)
	}

	if slices.Contains(firstList, "github_search_repos") {
		t.Errorf("backend tool should be hidden before load: %v", firstList)
	}

	// Second tools/list (id 4): the loaded tool is now visible.
	secondList := toolNames(byID[4])
	if !slices.Contains(secondList, "github_search_repos") {
		t.Errorf("loaded tool missing from tools/list after assern_load: %v", secondList)
	}

	// tools/call (id 5): routed to the backend, returns the mock result.
	callText := contentText(byID[5])
	if !strings.Contains(callText, "mock result for search_repos") {
		t.Errorf("loaded tool call not routed to backend; got %q", callText)
	}

	if calls := mock.GetToolCalls(); len(calls) != 1 || calls[0].Name != "search_repos" {
		t.Errorf("expected backend to receive search_repos call, got %+v", calls)
	}
}

func responsesByID(t *testing.T, out string) map[int]rpcResp {
	t.Helper()

	byID := make(map[int]rpcResp)

	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var resp rpcResp
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("unmarshal response line %q: %v", line, err)
		}

		if resp.ID != nil {
			byID[*resp.ID] = resp
		}
	}

	return byID
}

func toolNames(resp rpcResp) []string {
	names := make([]string, len(resp.Result.Tools))
	for i, tool := range resp.Result.Tools {
		names[i] = tool.Name
	}

	return names
}

func contentText(resp rpcResp) string {
	var b strings.Builder
	for _, c := range resp.Result.Content {
		b.WriteString(c.Text)
	}

	return b.String()
}
