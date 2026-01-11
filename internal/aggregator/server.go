package aggregator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/valksor/go-assern/internal/config"
)

// TransportType represents the type of MCP transport.
type TransportType string

const (
	TransportStdio TransportType = "stdio"
	TransportSSE   TransportType = "sse"
	TransportHTTP  TransportType = "http"
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
		return nil, fmt.Errorf("server %s must have either command (stdio) or url (sse/http)", name)
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
		return errors.New("server already started")
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
		s.client, err = client.NewStdioMCPClient(s.cfg.Command, s.env, s.cfg.Args...)
	case TransportSSE:
		s.client, err = client.NewSSEMCPClient(s.cfg.URL)
	case TransportHTTP:
		s.client, err = client.NewStreamableHttpClient(s.cfg.URL)
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
		_ = s.client.Close()

		return fmt.Errorf("initializing connection: %w", err)
	}

	s.started = true
	s.logger.Info("server started successfully")

	return nil
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
		return nil, errors.New("server not started")
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
		return nil, errors.New("server not started")
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
