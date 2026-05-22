package aggregator

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/testutil"
)

func newCodeModeAggregator(t *testing.T) *Aggregator {
	t.Helper()

	agg, err := New(Options{
		Config: &config.Config{Settings: &config.Settings{CodeMode: &config.CodeModeConfig{Enabled: true}}},
		Logger: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()

	mock := testutil.NewMockServer("github", []mcp.Tool{
		mcp.NewTool("ping", mcp.WithDescription("ping the server")),
		mcp.NewTool("echo", mcp.WithDescription("echo input")),
	})
	mock.SetToolResult("ping", mcp.NewToolResultText("pong"))
	mock.SetToolResult("echo", mcp.NewToolResultText("echoed"))

	if startErr := mock.Start(ctx); startErr != nil {
		t.Fatalf("mock.Start: %v", startErr)
	}

	if addErr := agg.AddServer(ctx, mock); addErr != nil {
		t.Fatalf("AddServer: %v", addErr)
	}

	agg.CreateMCPServer()

	return agg
}

func executeCode(code string) mcp.CallToolRequest {
	var req mcp.CallToolRequest
	req.Params.Name = ToolExecuteName
	req.Params.Arguments = map[string]any{"code": code}

	return req
}

func TestCodeModeToolRegistered(t *testing.T) {
	t.Parallel()

	agg := newCodeModeAggregator(t)

	sess := newFakeSession("cm-1")
	registerSession(t, agg.mcpServer, sess)

	if names := listToolNames(t, agg.mcpServer, sess); !slices.Contains(names, ToolExecuteName) {
		t.Errorf("assern_execute not exposed: %v", names)
	}
}

func TestHandleExecuteRoutesToBackend(t *testing.T) {
	t.Parallel()

	agg := newCodeModeAggregator(t)

	res, err := agg.handleExecute(context.Background(), executeCode(`print(call("github_ping", {}))`))
	if err != nil {
		t.Fatalf("handleExecute: %v", err)
	}

	if res.IsError {
		t.Fatalf("unexpected error result: %s", textContent(t, res))
	}

	if got := textContent(t, res); !strings.Contains(got, "pong") {
		t.Errorf("output %q does not contain backend result 'pong'", got)
	}
}

func TestHandleExecuteComposesTools(t *testing.T) {
	t.Parallel()

	agg := newCodeModeAggregator(t)

	code := `
a = call("github_ping", {})
b = call("github_echo", {})
print(a + "+" + b)
`

	res, err := agg.handleExecute(context.Background(), executeCode(code))
	if err != nil {
		t.Fatalf("handleExecute: %v", err)
	}

	if got := textContent(t, res); !strings.Contains(got, "pong+echoed") {
		t.Errorf("composed output %q missing 'pong+echoed'", got)
	}
}

func TestHandleExecuteRespectsAllowlist(t *testing.T) {
	t.Parallel()

	agg, err := New(Options{
		Config: &config.Config{Settings: &config.Settings{CodeMode: &config.CodeModeConfig{
			Enabled:      true,
			AllowedTools: []string{"github_ping"}, // echo is excluded
		}}},
		Logger: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	mock := testutil.NewMockServer("github", []mcp.Tool{
		mcp.NewTool("ping"), mcp.NewTool("echo"),
	})
	mock.SetToolResult("ping", mcp.NewToolResultText("pong"))

	if startErr := mock.Start(ctx); startErr != nil {
		t.Fatalf("start: %v", startErr)
	}

	if addErr := agg.AddServer(ctx, mock); addErr != nil {
		t.Fatalf("add: %v", addErr)
	}

	agg.CreateMCPServer()

	// Allowed tool works.
	if _, err := agg.callToolText(ctx, "github_ping", nil); err != nil {
		t.Errorf("allowed tool github_ping should be callable: %v", err)
	}

	// Excluded tool is blocked.
	_, err = agg.callToolText(ctx, "github_echo", nil)
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("github_echo should be blocked by allowlist, got %v", err)
	}

	// search() must not surface disallowed tools (no enumeration leak).
	matches := agg.searchMatches("", 0)
	for _, m := range matches {
		if m.Name == "github_echo" {
			t.Errorf("searchMatches leaked disallowed tool github_echo: %+v", matches)
		}
	}

	if len(matches) != 1 || matches[0].Name != "github_ping" {
		t.Errorf("searchMatches = %+v, want only github_ping", matches)
	}
}

func TestHandleExecuteUnknownToolErrors(t *testing.T) {
	t.Parallel()

	agg := newCodeModeAggregator(t)

	res, err := agg.handleExecute(context.Background(), executeCode(`call("github_missing", {})`))
	if err != nil {
		t.Fatalf("handleExecute: %v", err)
	}

	if !res.IsError {
		t.Fatalf("expected error result for unknown tool, got %s", textContent(t, res))
	}

	if got := textContent(t, res); !strings.Contains(got, "tool not found") {
		t.Errorf("error %q should mention 'tool not found'", got)
	}
}
