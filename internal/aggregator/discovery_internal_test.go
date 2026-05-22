package aggregator

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/valksor/go-assern/internal/config"
)

// fakeSession is a minimal ClientSession that supports per-session tools,
// used to exercise the discovery handlers without a real transport.
type fakeSession struct {
	id          string
	notes       chan mcp.JSONRPCNotification
	initialized atomic.Bool
	tools       map[string]server.ServerTool
}

func newFakeSession(id string) *fakeSession {
	s := &fakeSession{
		id:    id,
		notes: make(chan mcp.JSONRPCNotification, 100),
		tools: make(map[string]server.ServerTool),
	}
	s.initialized.Store(true)

	return s
}

func (s *fakeSession) SessionID() string                                   { return s.id }
func (s *fakeSession) NotificationChannel() chan<- mcp.JSONRPCNotification { return s.notes }
func (s *fakeSession) Initialize()                                         { s.initialized.Store(true) }

func (s *fakeSession) Initialized() bool                              { return s.initialized.Load() }
func (s *fakeSession) GetSessionTools() map[string]server.ServerTool  { return s.tools }
func (s *fakeSession) SetSessionTools(t map[string]server.ServerTool) { s.tools = t }

var _ server.SessionWithTools = (*fakeSession)(nil)

type toolSpec struct {
	server, name, desc string
}

func newDiscoveryAggregator(t *testing.T, dc *config.DiscoveryConfig, specs ...toolSpec) *Aggregator {
	t.Helper()

	agg := &Aggregator{
		cfg:       &config.Config{Settings: &config.Settings{Discovery: dc}},
		logger:    slog.New(slog.DiscardHandler),
		servers:   make(map[string]Server),
		tools:     NewToolRegistry(),
		resources: NewResourceRegistry(),
		prompts:   NewPromptRegistry(),
		health:    NewHealthTracker(DefaultHealthThreshold),
	}

	for _, s := range specs {
		agg.tools.Register(s.server, mcp.NewTool(s.name, mcp.WithDescription(s.desc)), nil)
	}

	return agg
}

func defaultSpecs() []toolSpec {
	return []toolSpec{
		{server: "github", name: "search_repos", desc: "Search repositories"},
		{server: "github", name: "create_issue", desc: "Open an issue"},
		{server: "linear", name: "search", desc: "Search tickets"},
	}
}

func loadReq(names ...string) mcp.CallToolRequest {
	anyNames := make([]any, len(names))
	for i, n := range names {
		anyNames[i] = n
	}

	var req mcp.CallToolRequest
	req.Params.Name = ToolLoadName
	req.Params.Arguments = map[string]any{"names": anyNames}

	return req
}

// listToolNames sends a tools/list request through the server for the given
// session and returns the tool names visible to that session.
func listToolNames(t *testing.T, srv *server.MCPServer, sess server.ClientSession) []string {
	t.Helper()

	ctx := srv.WithContext(context.Background(), sess)

	raw, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	if err != nil {
		t.Fatalf("marshal tools/list: %v", err)
	}

	resp := srv.HandleMessage(ctx, raw)

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	var parsed struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal tools/list response: %v", err)
	}

	names := make([]string, len(parsed.Result.Tools))
	for i, tool := range parsed.Result.Tools {
		names[i] = tool.Name
	}

	return names
}

func registerSession(t *testing.T, srv *server.MCPServer, sess server.ClientSession) {
	t.Helper()

	if err := srv.RegisterSession(context.Background(), sess); err != nil {
		t.Fatalf("RegisterSession: %v", err)
	}
}

func TestCreateMCPServerDiscoveryDisabledExposesAllTools(t *testing.T) {
	t.Parallel()

	agg := newDiscoveryAggregator(t, nil, defaultSpecs()...)
	srv := agg.CreateMCPServer()

	sess := newFakeSession("disabled-1")
	registerSession(t, srv, sess)

	names := listToolNames(t, srv, sess)

	for _, want := range []string{"github_search_repos", "github_create_issue", "linear_search"} {
		if !slices.Contains(names, want) {
			t.Errorf("disabled mode missing backend tool %q; got %v", want, names)
		}
	}

	if slices.Contains(names, ToolSearchName) {
		t.Errorf("disabled mode should not expose %s; got %v", ToolSearchName, names)
	}
}

func TestCreateMCPServerDiscoveryEnabledHidesBackendTools(t *testing.T) {
	t.Parallel()

	agg := newDiscoveryAggregator(t, &config.DiscoveryConfig{Enabled: true, Pinned: []string{"linear_search"}}, defaultSpecs()...)
	srv := agg.CreateMCPServer()

	sess := newFakeSession("enabled-1")
	registerSession(t, srv, sess)

	names := listToolNames(t, srv, sess)

	for _, want := range []string{ToolSearchName, ToolLoadName, ToolForgetName, "linear_search"} {
		if !slices.Contains(names, want) {
			t.Errorf("enabled mode missing %q; got %v", want, names)
		}
	}

	// Non-pinned backend tools must be hidden until loaded.
	if slices.Contains(names, "github_search_repos") {
		t.Errorf("enabled mode should hide github_search_repos until loaded; got %v", names)
	}
}

func TestHandleLoadAndForget(t *testing.T) {
	t.Parallel()

	agg := newDiscoveryAggregator(t, &config.DiscoveryConfig{Enabled: true}, defaultSpecs()...)
	srv := agg.CreateMCPServer()

	sess := newFakeSession("load-1")
	registerSession(t, srv, sess)
	ctx := srv.WithContext(context.Background(), sess)

	res, err := agg.handleLoad(ctx, loadReq("github_search_repos", "does_not_exist"))
	if err != nil {
		t.Fatalf("handleLoad error: %v", err)
	}

	if res.IsError {
		t.Fatalf("handleLoad returned error result: %+v", res)
	}

	if _, ok := sess.GetSessionTools()["github_search_repos"]; !ok {
		t.Errorf("github_search_repos not loaded into session: %v", sess.GetSessionTools())
	}

	if agg.discovery.loadedCount(sess.id) != 1 {
		t.Errorf("loadedCount = %d, want 1", agg.discovery.loadedCount(sess.id))
	}

	// Now forget it.
	var forget mcp.CallToolRequest
	forget.Params.Arguments = map[string]any{"names": []any{"github_search_repos"}}

	if _, err := agg.handleForget(ctx, forget); err != nil {
		t.Fatalf("handleForget error: %v", err)
	}

	if _, ok := sess.GetSessionTools()["github_search_repos"]; ok {
		t.Error("github_search_repos still loaded after forget")
	}

	if agg.discovery.loadedCount(sess.id) != 0 {
		t.Errorf("loadedCount after forget = %d, want 0", agg.discovery.loadedCount(sess.id))
	}
}

func TestHandleLoadEvictsOverCeiling(t *testing.T) {
	t.Parallel()

	agg := newDiscoveryAggregator(t, &config.DiscoveryConfig{Enabled: true, MaxLoaded: 2}, defaultSpecs()...)
	srv := agg.CreateMCPServer()

	sess := newFakeSession("evict-1")
	registerSession(t, srv, sess)
	ctx := srv.WithContext(context.Background(), sess)

	// Load three tools sequentially; the ceiling is 2.
	for _, name := range []string{"github_search_repos", "github_create_issue", "linear_search"} {
		if _, err := agg.handleLoad(ctx, loadReq(name)); err != nil {
			t.Fatalf("handleLoad(%s): %v", name, err)
		}
	}

	if got := agg.discovery.loadedCount(sess.id); got != 2 {
		t.Errorf("loadedCount = %d, want 2 (ceiling)", got)
	}

	// The first-loaded tool must have been evicted.
	if _, ok := sess.GetSessionTools()["github_search_repos"]; ok {
		t.Error("oldest tool github_search_repos should have been evicted")
	}

	if _, ok := sess.GetSessionTools()["linear_search"]; !ok {
		t.Error("most-recent tool linear_search should be loaded")
	}
}

func TestHandleLoadSingleBatchOverCeiling(t *testing.T) {
	t.Parallel()

	agg := newDiscoveryAggregator(t, &config.DiscoveryConfig{Enabled: true, MaxLoaded: 2}, defaultSpecs()...)
	srv := agg.CreateMCPServer()

	sess := newFakeSession("batch-1")
	registerSession(t, srv, sess)
	ctx := srv.WithContext(context.Background(), sess)

	// Load three tools in a SINGLE call; ceiling is 2. The session must end up
	// with exactly the ceiling, and the tracker must match the session.
	if _, err := agg.handleLoad(ctx, loadReq("github_search_repos", "github_create_issue", "linear_search")); err != nil {
		t.Fatalf("handleLoad: %v", err)
	}

	if got := agg.discovery.loadedCount(sess.id); got != 2 {
		t.Errorf("loadedCount = %d, want 2 (ceiling)", got)
	}

	if got := len(sess.GetSessionTools()); got != 2 {
		t.Errorf("session has %d tools, want 2; tracker and session diverged: %v", got, sess.GetSessionTools())
	}

	// The first-listed tool is the oldest and must have been evicted.
	if _, ok := sess.GetSessionTools()["github_search_repos"]; ok {
		t.Error("oldest tool in the batch should have been evicted from the session")
	}
}

func TestDiscoveryPerSessionIsolation(t *testing.T) {
	t.Parallel()

	agg := newDiscoveryAggregator(t, &config.DiscoveryConfig{Enabled: true}, defaultSpecs()...)
	srv := agg.CreateMCPServer()

	sessA := newFakeSession("iso-a")
	sessB := newFakeSession("iso-b")
	registerSession(t, srv, sessA)
	registerSession(t, srv, sessB)

	ctxA := srv.WithContext(context.Background(), sessA)
	if _, err := agg.handleLoad(ctxA, loadReq("github_search_repos")); err != nil {
		t.Fatalf("load into A: %v", err)
	}

	// Session B must not see A's loaded tool.
	if _, ok := sessB.GetSessionTools()["github_search_repos"]; ok {
		t.Error("session B leaked session A's loaded tool")
	}

	bNames := listToolNames(t, srv, sessB)
	if slices.Contains(bNames, "github_search_repos") {
		t.Errorf("session B tools/list leaked A's tool: %v", bNames)
	}
}

func TestHandleSearchReturnsRankedJSON(t *testing.T) {
	t.Parallel()

	agg := newDiscoveryAggregator(t, &config.DiscoveryConfig{Enabled: true}, defaultSpecs()...)
	_ = agg.CreateMCPServer()

	var req mcp.CallToolRequest
	req.Params.Arguments = map[string]any{"query": "search"}

	res, err := agg.handleSearch(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSearch error: %v", err)
	}

	text := textContent(t, res)

	var parsed struct {
		Count   int `json:"count"`
		Matches []struct {
			Name            string `json:"name"`
			EstimatedTokens int    `json:"estimated_tokens"`
		} `json:"matches"`
	}

	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("unmarshal search result %q: %v", text, err)
	}

	if parsed.Count == 0 || len(parsed.Matches) == 0 {
		t.Fatalf("expected search matches, got %s", text)
	}

	if parsed.Matches[0].EstimatedTokens <= 0 {
		t.Errorf("expected positive token estimate, got %d", parsed.Matches[0].EstimatedTokens)
	}
}

func textContent(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()

	if res == nil || len(res.Content) == 0 {
		t.Fatal("empty tool result")
	}

	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", res.Content[0])
	}

	return tc.Text
}
