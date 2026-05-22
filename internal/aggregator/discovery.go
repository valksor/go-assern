package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/valksor/go-assern/internal/config"
)

// Meta-tool names exposed in discovery mode. They use the reserved "assern_"
// prefix so they never collide with prefixed backend tools (server_tool).
const (
	ToolSearchName = "assern_search"
	ToolLoadName   = "assern_load"
	ToolForgetName = "assern_forget"
)

// discoveryState tracks per-session tool loading for progressive disclosure.
// Loaded tools are scoped to a single client session so that one client's
// discovery activity never changes the tool list seen by another.
type discoveryState struct {
	cfg *config.DiscoveryConfig

	mu    sync.Mutex
	loads map[string][]string // sessionID -> loaded prefixed names, oldest first
}

func newDiscoveryState(cfg *config.DiscoveryConfig) *discoveryState {
	return &discoveryState{
		cfg:   cfg,
		loads: make(map[string][]string),
	}
}

// recordLoads moves each name to the most-recent position of a session's load
// order and evicts the oldest entries beyond maxLoaded (0 = unlimited). It
// returns the evicted names so the caller can drop them from the session.
func (d *discoveryState) recordLoads(sessionID string, names []string, maxLoaded int) []string {
	d.mu.Lock()
	defer d.mu.Unlock()

	order := d.loads[sessionID]
	for _, n := range names {
		order = removeString(order, n)
		order = append(order, n)
	}

	var evicted []string

	if maxLoaded > 0 {
		for len(order) > maxLoaded {
			evicted = append(evicted, order[0])
			order = order[1:]
		}
	}

	d.loads[sessionID] = slices.Clone(order)

	return evicted
}

// removeLoads drops names from a session's load order.
func (d *discoveryState) removeLoads(sessionID string, names []string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	order := d.loads[sessionID]
	for _, n := range names {
		order = removeString(order, n)
	}

	if len(order) == 0 {
		delete(d.loads, sessionID)

		return
	}

	d.loads[sessionID] = order
}

// forgetSession discards all load state for a disconnected session.
func (d *discoveryState) forgetSession(sessionID string) {
	d.mu.Lock()
	delete(d.loads, sessionID)
	d.mu.Unlock()
}

// loadedCount returns how many tools a session currently has loaded.
func (d *discoveryState) loadedCount(sessionID string) int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return len(d.loads[sessionID])
}

// removeString returns s with the first occurrence of target removed, without
// mutating the backing array shared with other slices.
func removeString(s []string, target string) []string {
	if i := slices.Index(s, target); i >= 0 {
		return append(s[:i:i], s[i+1:]...)
	}

	return s
}

// DiscoveryEnabled reports whether progressive tool disclosure is active.
func (a *Aggregator) DiscoveryEnabled() bool {
	return a.discoveryConfig().IsEnabled()
}

// discoveryConfig returns the configured discovery settings, or nil. It reads
// a.cfg under cfgMu because Reload may swap a.cfg on another goroutine.
func (a *Aggregator) discoveryConfig() *config.DiscoveryConfig {
	a.cfgMu.RLock()
	defer a.cfgMu.RUnlock()

	if a.cfg == nil || a.cfg.Settings == nil {
		return nil
	}

	return a.cfg.Settings.Discovery
}

// discoveryHooks returns server hooks that clean up per-session discovery state
// when a client disconnects.
func (a *Aggregator) discoveryHooks() *server.Hooks {
	hooks := &server.Hooks{}
	hooks.OnUnregisterSession = append(hooks.OnUnregisterSession,
		func(_ context.Context, session server.ClientSession) {
			if a.discovery != nil {
				a.discovery.forgetSession(session.SessionID())
			}
		})

	return hooks
}

// registerMetaTools adds the assern_* discovery meta-tools to the MCP server.
func (a *Aggregator) registerMetaTools() {
	a.mcpServer.AddTool(mcp.NewTool(
		ToolSearchName,
		mcp.WithDescription("Search the catalog of available tools by keyword. Returns matching tool names with descriptions. A returned tool is NOT callable until loaded: you MUST call assern_load with its name before calling it."),
		mcp.WithString("query", mcp.Description("Free-text search over tool names, descriptions, and server names. An empty query lists the catalog.")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return.")),
	), a.handleSearch)

	a.mcpServer.AddTool(mcp.NewTool(
		ToolLoadName,
		mcp.WithDescription("Load one or more tools (by their prefixed name from assern_search) so they become callable in this session. Check the returned 'not_found' and 'failed' lists: any name listed there was NOT loaded and must not be called."),
		mcp.WithArray("names", mcp.Required(), mcp.Description("Prefixed tool names to load, e.g. github_search_repos."), mcp.WithStringItems()),
	), a.handleLoad)

	a.mcpServer.AddTool(mcp.NewTool(
		ToolForgetName,
		mcp.WithDescription("Unload tools previously loaded with assern_load, freeing context in this session."),
		mcp.WithArray("names", mcp.Required(), mcp.Description("Prefixed tool names to unload."), mcp.WithStringItems()),
	), a.handleForget)
}

// pinnedSet returns the configured pinned tool names as a set for quick lookup.
func (a *Aggregator) pinnedSet() map[string]struct{} {
	cfg := a.discoveryConfig()
	if cfg == nil {
		return nil
	}

	set := make(map[string]struct{}, len(cfg.Pinned))
	for _, name := range cfg.Pinned {
		set[name] = struct{}{}
	}

	return set
}

// exposePinnedTools adds always-on tools (from discovery.pinned) to the server.
func (a *Aggregator) exposePinnedTools() {
	cfg := a.discoveryConfig()
	if cfg == nil {
		return
	}

	for _, name := range cfg.Pinned {
		entry, ok := a.tools.Get(name)
		if !ok {
			a.logger.Warn("pinned tool not found in catalog", "tool", name)

			continue
		}

		a.addToolToServer(entry)
	}
}

// searchMatch is the JSON shape returned for each assern_search result.
type searchMatch struct {
	Name            string `json:"name"`
	Server          string `json:"server"`
	Description     string `json:"description,omitempty"`
	EstimatedTokens int    `json:"estimated_tokens"`
}

// handleSearch implements the assern_search meta-tool.
func (a *Aggregator) handleSearch(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")

	limit := req.GetInt("limit", 0)
	if limit <= 0 {
		limit = a.discoveryConfig().EffectiveMaxResults()
	}

	matches := a.tools.Search(query, limit)

	results := make([]searchMatch, 0, len(matches))
	for _, e := range matches {
		results = append(results, searchMatch{
			Name:            e.PrefixedName,
			Server:          e.ServerName,
			Description:     e.Tool.Description,
			EstimatedTokens: EstimateToolTokens(e.ExposedTool()),
		})
	}

	return jsonResult(map[string]any{
		"matches": results,
		"count":   len(results),
		"hint":    "Call assern_load with the names you need to make them callable.",
	}), nil
}

// handleLoad implements the assern_load meta-tool.
func (a *Aggregator) handleLoad(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	names, err := req.RequireStringSlice("names")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid 'names' argument: %v", err)), nil
	}

	sessionID, errResult := sessionID(ctx)
	if errResult != nil {
		return errResult, nil
	}

	var loaded, notFound, failed []string

	// Add every requested tool we can, collecting per-tool outcomes rather than
	// aborting mid-batch. Aborting would leave already-added tools live on the
	// session but untracked by discovery state (and so unbounded by max_loaded).
	for _, name := range names {
		entry, ok := a.tools.Get(name)
		if !ok {
			notFound = append(notFound, name)

			continue
		}

		if addErr := a.mcpServer.AddSessionTool(sessionID, entry.ExposedTool(), a.createToolHandler(entry)); addErr != nil {
			a.logger.Warn("failed to load session tool", "tool", name, "session", sessionID, "error", addErr)
			failed = append(failed, name)

			continue
		}

		loaded = append(loaded, entry.PrefixedName)
	}

	evicted := a.discovery.recordLoads(sessionID, loaded, a.discoveryConfig().EffectiveMaxLoaded())
	if len(evicted) > 0 {
		if delErr := a.mcpServer.DeleteSessionTools(sessionID, evicted...); delErr != nil {
			a.logger.Warn("failed to evict session tools", "session", sessionID, "error", delErr)
		}
	}

	return jsonResult(map[string]any{
		"loaded":    loaded,
		"not_found": notFound,
		"failed":    failed,
		"evicted":   evicted,
		"active":    a.discovery.loadedCount(sessionID),
	}), nil
}

// handleForget implements the assern_forget meta-tool.
func (a *Aggregator) handleForget(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	names, err := req.RequireStringSlice("names")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid 'names' argument: %v", err)), nil
	}

	sid, errResult := sessionID(ctx)
	if errResult != nil {
		return errResult, nil
	}

	if delErr := a.mcpServer.DeleteSessionTools(sid, names...); delErr != nil {
		a.logger.Warn("failed to unload session tools", "session", sid, "error", delErr)
	}

	a.discovery.removeLoads(sid, names)

	return jsonResult(map[string]any{
		"forgotten": names,
		"active":    a.discovery.loadedCount(sid),
	}), nil
}

// sessionID extracts the calling session's ID from the context, returning an
// error tool result when no session is present.
func sessionID(ctx context.Context) (string, *mcp.CallToolResult) {
	session := server.ClientSessionFromContext(ctx)
	if session == nil {
		return "", mcp.NewToolResultError("no active session; tool discovery requires a session-aware transport")
	}

	return session.SessionID(), nil
}

// jsonResult marshals payload to indented JSON and wraps it in a text result.
func jsonResult(payload any) *mcp.CallToolResult {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("encoding result: %v", err))
	}

	return mcp.NewToolResultText(string(data))
}
