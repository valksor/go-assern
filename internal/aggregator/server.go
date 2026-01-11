package aggregator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/valksor/go-assern/internal/config"
)

// TransportType represents the type of MCP transport.
type TransportType string

const (
	TransportStdio     TransportType = "stdio"
	TransportSSE       TransportType = "sse"
	TransportHTTP      TransportType = "http"
	TransportOAuthSSE  TransportType = "oauth-sse"
	TransportOAuthHTTP TransportType = "oauth-http"
	TransportInProcess TransportType = "in-process"
)

// ManagedServer represents a backend MCP server that Assern manages.
type ManagedServer struct {
	name          string
	cfg           *config.ServerConfig
	env           []string
	logger        *slog.Logger
	transportType TransportType

	client *client.Client

	mu      sync.RWMutex
	started bool
}

// detectTransport determines the transport type from config.
func detectTransport(cfg *config.ServerConfig) TransportType {
	// Explicit transport takes precedence
	if cfg.Transport != "" {
		return TransportType(cfg.Transport)
	}

	// Auto-detect OAuth transports when OAuth config is present
	if cfg.OAuth != nil && cfg.URL != "" {
		return TransportOAuthHTTP // Default OAuth to HTTP (modern MCP standard)
	}

	// Auto-detect based on which fields are set
	if cfg.URL != "" {
		return TransportHTTP // Default URL-based to Streamable HTTP (modern MCP standard)
	}

	if cfg.Command != "" {
		return TransportStdio
	}

	return ""
}

// NewManagedServer creates a new managed server instance.
func NewManagedServer(name string, cfg *config.ServerConfig, env []string, logger *slog.Logger) (*ManagedServer, error) {
	transportType := detectTransport(cfg)

	if transportType == "" {
		return nil, fmt.Errorf("server %s: %w", name, ErrInvalidTransport)
	}

	return &ManagedServer{
		name:          name,
		cfg:           cfg,
		env:           env,
		logger:        logger.With("server", name),
		transportType: transportType,
	}, nil
}

// Start initializes the backend server connection.
func (s *ManagedServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return ErrServerAlreadyStarted
	}

	s.logger.Debug("starting server",
		"transport", s.transportType,
		"command", s.cfg.Command,
		"url", s.cfg.URL,
	)

	// Create client based on transport type
	var err error

	switch s.transportType {
	case TransportStdio:
		s.client, err = s.createStdioClient()
	case TransportSSE:
		s.client, err = s.createSSEClient()
	case TransportHTTP:
		s.client, err = s.createHTTPClient()
	case TransportOAuthSSE:
		s.client, err = s.createOAuthSSEClient()
	case TransportOAuthHTTP:
		s.client, err = s.createOAuthHTTPClient()
	case TransportInProcess:
		return errors.New("in-process transport requires explicit server reference")
	default:
		return fmt.Errorf("unsupported transport type: %s", s.transportType)
	}

	if err != nil {
		return fmt.Errorf("creating %s client: %w", s.transportType, err)
	}

	// Start the client (required before Initialize)
	if err := s.client.Start(ctx); err != nil {
		return fmt.Errorf("starting %s client: %w", s.transportType, err)
	}

	// Initialize the connection
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "Valksor Assern",
		Version: "1.0.0",
	}
	initReq.Params.Capabilities = mcp.ClientCapabilities{}

	_, err = s.client.Initialize(ctx, initReq)
	if err != nil {
		if closeErr := s.client.Close(); closeErr != nil {
			s.logger.Warn("error closing client after init failure", "error", closeErr)
		}

		return fmt.Errorf("initializing connection: %w", err)
	}

	s.started = true
	s.logger.Info("server started successfully")

	return nil
}

// createStdioClient creates a stdio transport client.
func (s *ManagedServer) createStdioClient() (*client.Client, error) {
	return client.NewStdioMCPClient(s.cfg.Command, s.env, s.cfg.Args...)
}

// createSSEClient creates an SSE transport client with optional headers.
func (s *ManagedServer) createSSEClient() (*client.Client, error) {
	opts := []transport.ClientOption{}

	// Add custom headers if configured
	if len(s.cfg.Headers) > 0 {
		opts = append(opts, transport.WithHeaders(s.cfg.Headers))
	}

	return client.NewSSEMCPClient(s.cfg.URL, opts...)
}

// createHTTPClient creates a Streamable HTTP transport client with optional headers.
func (s *ManagedServer) createHTTPClient() (*client.Client, error) {
	opts := []transport.StreamableHTTPCOption{}

	// Add custom headers if configured
	if len(s.cfg.Headers) > 0 {
		opts = append(opts, transport.WithHTTPHeaders(s.cfg.Headers))
	}

	return client.NewStreamableHttpClient(s.cfg.URL, opts...)
}

// createOAuthSSEClient creates an SSE client with OAuth authentication.
func (s *ManagedServer) createOAuthSSEClient() (*client.Client, error) {
	if s.cfg.OAuth == nil {
		return nil, fmt.Errorf("oauth-sse transport: %w", ErrOAuthRequired)
	}

	oauthCfg := transport.OAuthConfig{
		ClientID:              s.cfg.OAuth.ClientID,
		ClientSecret:          s.cfg.OAuth.ClientSecret,
		RedirectURI:           s.cfg.OAuth.RedirectURI,
		Scopes:                s.cfg.OAuth.Scopes,
		AuthServerMetadataURL: s.cfg.OAuth.AuthServerMetadataURL,
		PKCEEnabled:           s.cfg.OAuth.PKCEEnabled,
	}

	opts := []transport.ClientOption{}

	// Add additional headers if configured
	if len(s.cfg.Headers) > 0 {
		opts = append(opts, transport.WithHeaders(s.cfg.Headers))
	}

	return client.NewOAuthSSEClient(s.cfg.URL, oauthCfg, opts...)
}

// createOAuthHTTPClient creates a Streamable HTTP client with OAuth authentication.
func (s *ManagedServer) createOAuthHTTPClient() (*client.Client, error) {
	if s.cfg.OAuth == nil {
		return nil, fmt.Errorf("oauth-http transport: %w", ErrOAuthRequired)
	}

	oauthCfg := transport.OAuthConfig{
		ClientID:              s.cfg.OAuth.ClientID,
		ClientSecret:          s.cfg.OAuth.ClientSecret,
		RedirectURI:           s.cfg.OAuth.RedirectURI,
		Scopes:                s.cfg.OAuth.Scopes,
		AuthServerMetadataURL: s.cfg.OAuth.AuthServerMetadataURL,
		PKCEEnabled:           s.cfg.OAuth.PKCEEnabled,
	}

	opts := []transport.StreamableHTTPCOption{}

	// Add additional headers if configured
	if len(s.cfg.Headers) > 0 {
		opts = append(opts, transport.WithHTTPHeaders(s.cfg.Headers))
	}

	return client.NewOAuthStreamableHttpClient(s.cfg.URL, oauthCfg, opts...)
}

// Stop gracefully shuts down the server connection.
func (s *ManagedServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	s.logger.Debug("stopping server")

	if s.client != nil {
		if err := s.client.Close(); err != nil {
			s.logger.Warn("error closing client", "error", err)
		}
	}

	s.started = false
	s.logger.Info("server stopped")

	return nil
}

// DiscoverTools queries the backend server for available tools.
func (s *ManagedServer) DiscoverTools(ctx context.Context) ([]mcp.Tool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return nil, ErrServerNotStarted
	}

	result, err := s.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("listing tools: %w", err)
	}

	s.logger.Debug("discovered tools", "count", len(result.Tools))

	return result.Tools, nil
}

// CallTool executes a tool on the backend server.
func (s *ManagedServer) CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return nil, ErrServerNotStarted
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	s.logger.Debug("calling tool", "name", name)

	result, err := s.client.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("calling tool %s: %w", name, err)
	}

	return result, nil
}

// Name returns the server name.
func (s *ManagedServer) Name() string {
	return s.name
}

// IsStarted returns whether the server is currently running.
func (s *ManagedServer) IsStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.started
}

// Config returns the server configuration.
func (s *ManagedServer) Config() *config.ServerConfig {
	return s.cfg
}

// DiscoverResources queries the backend server for available resources.
func (s *ManagedServer) DiscoverResources(ctx context.Context) ([]mcp.Resource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return nil, ErrServerNotStarted
	}

	result, err := s.client.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		return nil, fmt.Errorf("listing resources: %w", err)
	}

	s.logger.Debug("discovered resources", "count", len(result.Resources))

	return result.Resources, nil
}

// ReadResource reads a resource from the backend server.
func (s *ManagedServer) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return nil, ErrServerNotStarted
	}

	req := mcp.ReadResourceRequest{}
	req.Params.URI = uri

	s.logger.Debug("reading resource", "uri", uri)

	result, err := s.client.ReadResource(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("reading resource %s: %w", uri, err)
	}

	return result, nil
}

// DiscoverPrompts queries the backend server for available prompts.
func (s *ManagedServer) DiscoverPrompts(ctx context.Context) ([]mcp.Prompt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return nil, ErrServerNotStarted
	}

	result, err := s.client.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		return nil, fmt.Errorf("listing prompts: %w", err)
	}

	s.logger.Debug("discovered prompts", "count", len(result.Prompts))

	return result.Prompts, nil
}

// GetPrompt retrieves a prompt from the backend server.
func (s *ManagedServer) GetPrompt(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return nil, ErrServerNotStarted
	}

	req := mcp.GetPromptRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	s.logger.Debug("getting prompt", "name", name)

	result, err := s.client.GetPrompt(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("getting prompt %s: %w", name, err)
	}

	return result, nil
}
