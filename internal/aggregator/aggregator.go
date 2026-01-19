// Package aggregator provides the core MCP server aggregation functionality.
package aggregator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/toon-format/toon-go"

	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-toolkit/env"
	"github.com/valksor/go-toolkit/project"
	"github.com/valksor/go-toolkit/version"
)

// Aggregator combines multiple MCP servers into a single unified interface.
type Aggregator struct {
	cfg          *config.Config
	projectCtx   *project.Context
	envLoader    *env.Loader
	logger       *slog.Logger
	outputFormat string // "json" or "toon"

	servers   map[string]Server
	tools     *ToolRegistry
	resources *ResourceRegistry
	prompts   *PromptRegistry
	mu        sync.RWMutex

	mcpServer *server.MCPServer
}

// Options configures the aggregator.
type Options struct {
	Config       *config.Config
	Project      *project.Context
	EnvLoader    *env.Loader
	Logger       *slog.Logger
	Timeout      time.Duration
	OutputFormat string // "json" or "toon"
}

// New creates a new aggregator with the given options.
func New(opts Options) (*Aggregator, error) {
	if opts.Config == nil {
		return nil, ErrConfigRequired
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
		servers:      make(map[string]Server),
		tools:        NewToolRegistry(),
		resources:    NewResourceRegistry(),
		prompts:      NewPromptRegistry(),
	}

	return agg, nil
}

// Start initializes all configured servers and discovers their tools.
func (a *Aggregator) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	effectiveServers := config.GetEffectiveServers(a.cfg)
	if len(effectiveServers) == 0 {
		return fmt.Errorf("%w\n\nAdd servers to:\n  Global: ~/.valksor/assern/mcp.json\n  Local:  .assern/mcp.json (project-specific)\n\nRun 'assern config init' to create default config", ErrNoServers)
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
			return fmt.Errorf("%w: %d servers failed", ErrAllServersFailed, len(errs))
		}

		// Partial success - log warning but continue with details
		failedNames := make([]string, 0, len(errs))
		for _, err := range errs {
			failedNames = append(failedNames, err.Error())
		}
		a.logger.Warn(fmt.Sprintf("%d of %d servers started (%d failed)",
			len(a.servers), len(effectiveServers), len(errs)),
			"failed", failedNames,
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

	a.servers = make(map[string]Server)
	a.tools = NewToolRegistry()
	a.resources = NewResourceRegistry()
	a.prompts = NewPromptRegistry()

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	return nil
}

// CreateMCPServer creates the MCP server that exposes aggregated tools, resources, and prompts.
func (a *Aggregator) CreateMCPServer() *server.MCPServer {
	a.mcpServer = server.NewMCPServer(
		"Valksor Assern",
		version.Version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false), // subscribe=true, listChanged=false
		server.WithPromptCapabilities(false),         // listChanged=false
		server.WithLogging(),
	)

	// Add all registered tools
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, entry := range a.tools.All() {
		a.addToolToServer(entry)
	}

	// Add all registered resources
	for _, entry := range a.resources.All() {
		a.addResourceToServer(entry)
	}

	// Add all registered prompts
	for _, entry := range a.prompts.All() {
		a.addPromptToServer(entry)
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
			return mcp.NewToolResultError(fmt.Sprintf("%s: %v", entry.ServerName, ErrServerNotFound)), nil
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

// addResourceToServer adds a resource entry to the MCP server.
func (a *Aggregator) addResourceToServer(entry *ResourceEntry) {
	// Create a copy of the resource with prefixed URI
	prefixedResource := mcp.NewResource(
		entry.PrefixedURI,
		entry.Resource.Name,
		mcp.WithResourceDescription(entry.Resource.Description),
	)

	if entry.Resource.MIMEType != "" {
		prefixedResource.MIMEType = entry.Resource.MIMEType
	}

	// Create handler that routes to the backend server
	handler := a.createResourceHandler(entry)

	a.mcpServer.AddResource(prefixedResource, handler)
}

// createResourceHandler creates a handler function for a resource that routes to the backend.
func (a *Aggregator) createResourceHandler(entry *ResourceEntry) server.ResourceHandlerFunc {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		a.mu.RLock()
		srv, exists := a.servers[entry.ServerName]
		a.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("%s: %w", entry.ServerName, ErrServerNotFound)
		}

		// Check if server supports resources
		resourceSrv, ok := srv.(ResourceServer)
		if !ok {
			return nil, fmt.Errorf("server %s does not support resources", entry.ServerName)
		}

		// Route the read to the backend server with the original URI
		result, err := resourceSrv.ReadResource(ctx, entry.OriginalURI)
		if err != nil {
			return nil, fmt.Errorf("reading resource: %w", err)
		}

		return result.Contents, nil
	}
}

// addPromptToServer adds a prompt entry to the MCP server.
func (a *Aggregator) addPromptToServer(entry *PromptEntry) {
	// Create a copy of the prompt with prefixed name
	prefixedPrompt := mcp.Prompt{
		Name:        entry.PrefixedName,
		Description: entry.Prompt.Description,
		Arguments:   entry.Prompt.Arguments,
	}

	// Create handler that routes to the backend server
	handler := a.createPromptHandler(entry)

	a.mcpServer.AddPrompt(prefixedPrompt, handler)
}

// createPromptHandler creates a handler function for a prompt that routes to the backend.
func (a *Aggregator) createPromptHandler(entry *PromptEntry) server.PromptHandlerFunc {
	return func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		a.mu.RLock()
		srv, exists := a.servers[entry.ServerName]
		a.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("%s: %w", entry.ServerName, ErrServerNotFound)
		}

		// Check if server supports prompts
		promptSrv, ok := srv.(PromptServer)
		if !ok {
			return nil, fmt.Errorf("server %s does not support prompts", entry.ServerName)
		}

		// Route the get to the backend server with the original prompt name
		result, err := promptSrv.GetPrompt(ctx, entry.Prompt.Name, req.Params.Arguments)
		if err != nil {
			return nil, fmt.Errorf("getting prompt: %w", err)
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

	items := make([]map[string]any, 0, len(result.Content))
	for _, content := range result.Content {
		items = append(items, contentItemToMap(content))
	}

	data["content"] = items

	// Add metadata
	data["metadata"] = map[string]any{
		"format":       "toon",
		"contentCount": len(items),
	}

	return data
}

// contentItemToMap converts an MCP content item to a map for TOON encoding.
func contentItemToMap(content mcp.Content) map[string]any {
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
		item["type"] = "unknown"
		item["data"] = fmt.Sprintf("%v", c)
	}

	return item
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

// GetServer returns a server by name.
func (a *Aggregator) GetServer(name string) (Server, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	srv, ok := a.servers[name]

	return srv, ok
}

// AddServer adds a pre-created server to the aggregator.
// This is primarily useful for testing with mock servers.
// The server must already be started; this method will discover its tools, resources, and prompts.
func (a *Aggregator) AddServer(ctx context.Context, srv Server) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	name := srv.Name()
	if _, exists := a.servers[name]; exists {
		return fmt.Errorf("server %s already exists", name)
	}

	// Discover tools from the server
	tools, err := srv.DiscoverTools(ctx)
	if err != nil {
		return fmt.Errorf("discovering tools from %s: %w", name, err)
	}

	// Get allowed list from config if available
	var allowed []string
	if srv.Config() != nil {
		allowed = srv.Config().Allowed
	}

	// Register tools with prefix
	for _, tool := range tools {
		a.tools.Register(name, tool, allowed)
	}

	// Try to discover resources if server supports them
	var resourceCount int
	if resourceSrv, ok := srv.(ResourceServer); ok {
		resources, err := resourceSrv.DiscoverResources(ctx)
		if err != nil {
			a.logger.Debug("server does not provide resources", "server", name, "error", err)
		} else {
			for _, resource := range resources {
				a.resources.Register(name, resource)
			}
			resourceCount = len(resources)
		}
	}

	// Try to discover prompts if server supports them
	var promptCount int
	if promptSrv, ok := srv.(PromptServer); ok {
		prompts, err := promptSrv.DiscoverPrompts(ctx)
		if err != nil {
			a.logger.Debug("server does not provide prompts", "server", name, "error", err)
		} else {
			for _, prompt := range prompts {
				a.prompts.Register(name, prompt)
			}
			promptCount = len(prompts)
		}
	}

	a.servers[name] = srv
	a.logger.Info("server added", "name", name, "tools", len(tools), "resources", resourceCount, "prompts", promptCount)

	return nil
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
