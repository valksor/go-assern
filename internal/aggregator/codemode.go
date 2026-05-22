package aggregator

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/valksor/go-assern/internal/codemode"
	"github.com/valksor/go-assern/internal/config"
)

// ToolExecuteName is the meta-tool that runs a sandboxed Starlark script which
// can orchestrate several aggregated tools in one call.
const ToolExecuteName = "assern_execute"

// codeModeConfig returns the configured code-mode settings, or nil. It reads
// a.cfg under cfgMu because Reload may swap a.cfg on another goroutine.
func (a *Aggregator) codeModeConfig() *config.CodeModeConfig {
	a.cfgMu.RLock()
	defer a.cfgMu.RUnlock()

	if a.cfg == nil || a.cfg.Settings == nil {
		return nil
	}

	return a.cfg.Settings.CodeMode
}

// CodeModeEnabled reports whether the assern_execute meta-tool is active.
func (a *Aggregator) CodeModeEnabled() bool {
	return a.codeModeConfig().IsEnabled()
}

// registerExecuteTool adds the assern_execute meta-tool to the MCP server.
func (a *Aggregator) registerExecuteTool() {
	a.mcpServer.AddTool(mcp.NewTool(
		ToolExecuteName,
		mcp.WithDescription(
			"Run a sandboxed Starlark script that orchestrates multiple tools in one call. "+
				"Available builtins: search(query[, limit]) returns matching tools; "+
				"call(name, args) invokes a tool by its prefixed name and returns its text result. "+
				"Use print(...) to emit results. No file or network access.",
		),
		mcp.WithString("code", mcp.Required(), mcp.Description("The Starlark script to execute.")),
	), a.handleExecute)
}

// handleExecute implements the assern_execute meta-tool.
func (a *Aggregator) handleExecute(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	code, err := req.RequireString("code")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid 'code' argument: %v", err)), nil
	}

	cfg := a.codeModeConfig()

	executor := codemode.New(codemode.Options{
		Call:      a.callToolText,
		Search:    a.searchMatches,
		Timeout:   cfg.Timeout,
		MaxCalls:  cfg.MaxToolCalls,
		MaxOutput: cfg.MaxOutputBytes,
	})

	result, runErr := executor.Run(ctx, code)
	if runErr != nil {
		// Surface the error together with any output produced before it failed.
		msg := runErr.Error()
		if result.Output != "" {
			msg = msg + "\n--- output ---\n" + result.Output
		}

		return mcp.NewToolResultError(msg), nil
	}

	out := result.Output
	if strings.TrimSpace(out) == "" {
		out = "(script produced no output; use print(...) to emit results)"
	}

	return mcp.NewToolResultText(out), nil
}

// callToolText routes a tool call (by prefixed name or alias) to its backend
// server and returns the result as text. It is the bridge used by code mode.
func (a *Aggregator) callToolText(ctx context.Context, name string, args map[string]any) (string, error) {
	entry, ok := a.tools.Get(name)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	if !a.codeModeToolAllowed(entry.PrefixedName) {
		return "", fmt.Errorf("%w: %s", ErrToolNotAllowed, entry.PrefixedName)
	}

	a.mu.RLock()
	srv, exists := a.servers[entry.ServerName]
	a.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("%s: %w", entry.ServerName, ErrServerNotFound)
	}

	result, err := srv.CallTool(ctx, entry.Tool.Name, args)
	if err != nil {
		a.health.RecordFailure(entry.ServerName)

		return "", fmt.Errorf("%s: %w", entry.ServerName, err)
	}

	a.health.RecordSuccess(entry.ServerName)

	return toolResultText(result), nil
}

// codeModeToolAllowed reports whether a tool may be invoked from code mode.
// An empty allow-list permits any aggregated tool.
func (a *Aggregator) codeModeToolAllowed(prefixedName string) bool {
	cfg := a.codeModeConfig()
	if cfg == nil || len(cfg.AllowedTools) == 0 {
		return true
	}

	return slices.Contains(cfg.AllowedTools, prefixedName)
}

// searchMatches adapts the catalog search to the code-mode search builtin.
// When an allow-list is configured, only allowed tools are returned so a script
// cannot enumerate the names and descriptions of tools it may not call.
func (a *Aggregator) searchMatches(query string, limit int) []codemode.ToolMatch {
	if limit <= 0 {
		limit = config.DefaultDiscoveryMaxResults
	}

	cfg := a.codeModeConfig()
	restricted := cfg != nil && len(cfg.AllowedTools) > 0

	// When restricted, search the whole catalog and filter, so the limit
	// applies to allowed results rather than being consumed by disallowed ones.
	searchLimit := limit
	if restricted {
		searchLimit = 0
	}

	entries := a.tools.Search(query, searchLimit)

	matches := make([]codemode.ToolMatch, 0, limit)
	for _, e := range entries {
		if restricted && !slices.Contains(cfg.AllowedTools, e.PrefixedName) {
			continue
		}

		matches = append(matches, codemode.ToolMatch{
			Name:        e.PrefixedName,
			Server:      e.ServerName,
			Description: e.Tool.Description,
		})

		if len(matches) >= limit {
			break
		}
	}

	return matches
}

// toolResultText extracts the textual content from a tool result, joining
// multiple text blocks with newlines.
func toolResultText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}

	var parts []string

	for _, content := range result.Content {
		if tc, ok := content.(mcp.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}

	return strings.Join(parts, "\n")
}
