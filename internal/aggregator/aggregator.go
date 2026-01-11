// Package aggregator provides the core MCP server aggregation functionality.
package aggregator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/toon-format/toon-go"

	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/project"
	"github.com/valksor/go-assern/internal/version"
)

// Aggregator combines multiple MCP servers into a single unified interface.
type Aggregator struct {
	cfg          *config.Config
	projectCtx   *project.Context
	envLoader    *project.EnvLoader
	logger       *slog.Logger
	outputFormat string // "json" or "toon"

	servers map[string]*ManagedServer
	tools   *ToolRegistry
	mu      sync.RWMutex

	mcpServer *server.MCPServer
}

// Options configures the aggregator.
type Options struct {
	Config       *config.Config
	Project      *project.Context
	EnvLoader    *project.EnvLoader
	Logger       *slog.Logger
	Timeout      time.Duration
	OutputFormat string // "json" or "toon"
}

// New creates a new aggregator with the given options.
func New(opts Options) (*Aggregator, error) {
	if opts.Config == nil {
		return nil, errors.New("config is required")
	}

	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	if opts.Timeout == 0 {
		opts.Timeout = 60 * time.Second
	}

	// Default output format to JSON if not specified
	if opts.OutputFormat == "" {
		opts.OutputFormat = "json"
	}

	agg := &Aggregator{
		cfg:          opts.Config,
		projectCtx:   opts.Project,
		envLoader:    opts.EnvLoader,
		logger:       opts.Logger,
		outputFormat: opts.OutputFormat,
		servers:      make(map[string]*ManagedServer),
		tools:        NewToolRegistry(),
	}

	return agg, nil
}

// Start initializes all configured servers and discovers their tools.
func (a *Aggregator) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	effectiveServers := config.GetEffectiveServers(a.cfg)
	if len(effectiveServers) == 0 {
		return errors.New("no servers configured: check mcp.json exists at ~/.valksor/assern/mcp.json or .assern/mcp.json")
	}

	a.logger.Info("starting aggregator", "servers", len(effectiveServers))

	// Start each backend server
	var wg sync.WaitGroup

	errCh := make(chan error, len(effectiveServers))

	for name, srvCfg := range effectiveServers {
		wg.Add(1)

		go func(name string, cfg *config.ServerConfig) {
			defer wg.Done()

			if err := a.startServer(ctx, name, cfg); err != nil {
				errCh <- fmt.Errorf("server %s: %w", name, err)
			}
		}(name, srvCfg)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		for _, err := range errs {
			a.logger.Error("failed to start server", "error", err)
		}

		// If ALL servers failed, return error
		if len(a.servers) == 0 {
			return fmt.Errorf("all %d servers failed to start: %v", len(errs), errs)
		}

		// Partial success - log warning but continue
		a.logger.Warn("started with partial failures",
			"success", len(a.servers),
			"failed", len(errs),
		)
	}

	a.logger.Info("aggregator started",
		"active_servers", len(a.servers),
		"total_tools", a.tools.Count(),
	)

	if a.tools.Count() == 0 {
		a.logger.Warn("no tools registered - check server configurations and 'allowed' filters")
	}

	return nil
}

// startServer starts a single backend server and discovers its tools.
func (a *Aggregator) startServer(ctx context.Context, name string, cfg *config.ServerConfig) error {
	// Build environment for the server
	var env []string
	if a.envLoader != nil {
		projectName := ""
		if a.projectCtx != nil {
			projectName = a.projectCtx.Name
		}

		env = a.envLoader.BuildServerEnv(cfg.Env, projectName)
	}

	// Create managed server
	managed, err := NewManagedServer(name, cfg, env, a.logger)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	// Start and initialize the server
	if err := managed.Start(ctx); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	// Discover tools
	tools, err := managed.DiscoverTools(ctx)
	if err != nil {
		if stopErr := managed.Stop(); stopErr != nil {
			a.logger.Warn("error stopping server after discovery failure", "server", name, "error", stopErr)
		}

		return fmt.Errorf("discovering tools: %w", err)
	}

	// Register tools with prefix
	for _, tool := range tools {
		a.tools.Register(name, tool, cfg.Allowed)
	}

	a.servers[name] = managed
	a.logger.Info("server started", "name", name, "tools", len(tools))

	return nil
}

// Stop gracefully shuts down all backend servers.
func (a *Aggregator) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logger.Info("stopping aggregator")

	var errs []error

	for name, srv := range a.servers {
		if err := srv.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stopping %s: %w", name, err))
		}
	}

	a.servers = make(map[string]*ManagedServer)
	a.tools = NewToolRegistry()

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	return nil
}

// CreateMCPServer creates the MCP server that exposes aggregated tools.
func (a *Aggregator) CreateMCPServer() *server.MCPServer {
	a.mcpServer = server.NewMCPServer(
		"Valksor Assern",
		version.Version,
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	// Add all registered tools
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, entry := range a.tools.All() {
		a.addToolToServer(entry)
	}

	return a.mcpServer
}

// addToolToServer adds a tool entry to the MCP server.
func (a *Aggregator) addToolToServer(entry *ToolEntry) {
	// Create a copy of the tool with prefixed name
	prefixedTool := mcp.Tool{
		Name:        entry.PrefixedName,
		Description: entry.Tool.Description,
		InputSchema: entry.Tool.InputSchema,
	}

	// Create handler that routes to the backend server
	handler := a.createToolHandler(entry)

	a.mcpServer.AddTool(prefixedTool, handler)
}

// createToolHandler creates a handler function for a tool that routes to the backend.
func (a *Aggregator) createToolHandler(entry *ToolEntry) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		a.mu.RLock()
		srv, exists := a.servers[entry.ServerName]
		a.mu.RUnlock()

		if !exists {
			return mcp.NewToolResultError(fmt.Sprintf("server %s not available", entry.ServerName)), nil
		}

		// Route the call to the backend server with the original tool name
		args, ok := req.Params.Arguments.(map[string]any)
		if !ok && req.Params.Arguments != nil {
			return mcp.NewToolResultError("invalid arguments format"), nil
		}

		result, err := srv.CallTool(ctx, entry.Tool.Name, args)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("tool call failed: %v", err)), nil
		}

		// Format result as TOON if enabled
		if a.outputFormat == "toon" {
			toonResult, toonErr := a.formatAsTOON(result)
			if toonErr != nil {
				a.logger.Warn("failed to format result as TOON, using original", "error", toonErr)

				return result, nil // Fall back to original
			}

			return toonResult, nil
		}

		return result, nil
	}
}

// formatAsTOON converts a CallToolResult to TOON format.
func (a *Aggregator) formatAsTOON(result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
	if result == nil {
		return &mcp.CallToolResult{}, nil
	}

	data := a.extractContentData(result)

	toonBytes, err := toon.Marshal(data,
		toon.WithLengthMarkers(true),
		toon.WithIndent(2),
	)
	if err != nil {
		return nil, fmt.Errorf("TOON marshal failed: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(toonBytes),
			},
		},
		IsError: result.IsError,
	}, nil
}

// extractContentData converts MCP content to a map structure for TOON encoding.
func (a *Aggregator) extractContentData(result *mcp.CallToolResult) map[string]any {
	data := make(map[string]any)

	if result.IsError {
		data["error"] = true
	}

	// Extract content items
	var items []map[string]any
	for _, content := range result.Content {
		item := make(map[string]any)

		switch c := content.(type) {
		case mcp.TextContent:
			item["type"] = "text"
			item["text"] = c.Text
		case mcp.ImageContent:
			item["type"] = "image"
			item["data"] = c.Data
			item["mimeType"] = c.MIMEType
		default:
			// For unknown content types, store as string representation
			item["type"] = "unknown"
			item["data"] = fmt.Sprintf("%v", c)
		}

		items = append(items, item)
	}

	data["content"] = items

	// Add metadata
	data["metadata"] = map[string]any{
		"format":       "toon",
		"contentCount": len(items),
	}

	return data
}

// ListTools returns all available tools.
func (a *Aggregator) ListTools() []ToolEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entries := a.tools.All()
	result := make([]ToolEntry, len(entries))

	for i, e := range entries {
		result[i] = *e
	}

	return result
}

// GetServer returns a managed server by name.
func (a *Aggregator) GetServer(name string) (*ManagedServer, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	srv, ok := a.servers[name]

	return srv, ok
}

// ServerNames returns the names of all active servers.
func (a *Aggregator) ServerNames() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	names := make([]string, 0, len(a.servers))
	for name := range a.servers {
		names = append(names, name)
	}

	return names
}

// ProjectName returns the current project context name.
func (a *Aggregator) ProjectName() string {
	if a.projectCtx == nil {
		return ""
	}

	return a.projectCtx.Name
}
