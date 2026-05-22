package aggregator

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/version"
)

// CreateMCPServer creates the MCP server that exposes aggregated tools, resources, and prompts.
//
// When discovery is enabled, only the assern_* meta-tools (plus any pinned
// tools) are exposed up front; clients pull in the rest at runtime per session.
// When disabled, every aggregated tool is exposed, preserving the original
// behaviour.
func (a *Aggregator) CreateMCPServer() *server.MCPServer {
	discovery := a.DiscoveryEnabled()
	codeMode := a.CodeModeEnabled()

	// Initialise discovery state before taking the read lock so the field is
	// not written while a lock-holding reader could observe it. Safe to do
	// unlocked here: CreateMCPServer runs once at startup, before any session
	// (and thus any OnUnregisterSession hook) can exist.
	if discovery {
		a.discovery = newDiscoveryState(a.discoveryConfig())
	}

	if codeMode {
		if cfg := a.codeModeConfig(); cfg != nil && len(cfg.AllowedTools) == 0 {
			a.logger.Warn("code mode enabled with no allowed_tools: scripts may call any aggregated tool")
		}
	}

	opts := []server.ServerOption{
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false), // subscribe=true, listChanged=false
		server.WithPromptCapabilities(false),         // listChanged=false
		server.WithLogging(),
	}

	if discovery {
		opts = append(opts, server.WithHooks(a.discoveryHooks()))
	}

	a.mcpServer = server.NewMCPServer("Valksor Assern", version.Version, opts...)

	a.mu.RLock()
	defer a.mu.RUnlock()

	if discovery {
		a.registerMetaTools()
		a.exposePinnedTools()
	} else {
		// Add all registered tools.
		for _, entry := range a.tools.All() {
			a.addToolToServer(entry)
		}
	}

	// Code mode is independent of discovery: it adds one more meta-tool.
	if codeMode {
		a.registerExecuteTool()
	}

	// Resources and prompts are always exposed in full.
	for _, entry := range a.resources.All() {
		a.addResourceToServer(entry)
	}

	for _, entry := range a.prompts.All() {
		a.addPromptToServer(entry)
	}

	return a.mcpServer
}

// addToolToServer adds a tool entry to the MCP server.
func (a *Aggregator) addToolToServer(entry *ToolEntry) {
	// Create handler that routes to the backend server
	handler := a.createToolHandler(entry)

	a.mcpServer.AddTool(entry.ExposedTool(), handler)
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

		// Get retry config from server config
		var retryCfg *config.RetryConfig
		if cfg := srv.Config(); cfg != nil {
			retryCfg = cfg.Retry
		}

		// Execute with retry logic
		result, err := WithRetry(ctx, retryCfg, func(ctx context.Context, attempt int) (*mcp.CallToolResult, error) {
			if attempt > 1 {
				a.logger.Debug(
					"retrying tool call",
					"tool", entry.PrefixedName,
					"server", entry.ServerName,
					"attempt", attempt,
				)
			}

			return srv.CallTool(ctx, entry.Tool.Name, args)
		})
		if err != nil {
			a.health.RecordFailure(entry.ServerName)

			return mcp.NewToolResultError(fmt.Sprintf("tool call failed: %v", err)), nil
		}

		a.health.RecordSuccess(entry.ServerName)

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
