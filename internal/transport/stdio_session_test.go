package transport

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestStdioSessionImplementsToolInterface(t *testing.T) {
	t.Parallel()

	// Compile-time assertions already cover this; assert behaviour too.
	s := newStdioSession()

	s.SetSessionTools(map[string]server.ServerTool{
		"x": {Tool: mcp.NewTool("x")},
	})

	if _, ok := s.GetSessionTools()["x"]; !ok {
		t.Error("SetSessionTools/GetSessionTools round-trip failed")
	}
}

func TestRunSessionLoopHandlesRequests(t *testing.T) {
	t.Parallel()

	srv := server.NewMCPServer("test", "1.0.0", server.WithToolCapabilities(true))
	srv.AddTool(
		mcp.NewTool("ping", mcp.WithDescription("ping the server")),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("pong"), nil
		},
	)

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ping","arguments":{}}}`,
	}, "\n") + "\n"

	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	err := runSessionLoop(context.Background(), srv, newStdioSession(), strings.NewReader(input), &buf, logger)
	if err != nil {
		t.Fatalf("runSessionLoop error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, `"ping"`) {
		t.Errorf("tools/list response missing ping tool:\n%s", out)
	}

	if !strings.Contains(out, "pong") {
		t.Errorf("tools/call response missing pong:\n%s", out)
	}

	// Responses must be newline-delimited (one JSON object per line).
	lines := 0

	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) != "" {
			lines++
		}
	}

	if lines < 3 {
		t.Errorf("expected at least 3 response lines (init, list, call), got %d:\n%s", lines, out)
	}
}
