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

// ManagedServer represents a backend MCP server that Assern manages.
type ManagedServer struct {
	name   string
	cfg    *config.ServerConfig
	env    []string
	logger *slog.Logger

	client    *client.Client
	transport *transport.Stdio

	mu      sync.RWMutex
	started bool
}

// NewManagedServer creates a new managed server instance.
func NewManagedServer(name string, cfg *config.ServerConfig, env []string, logger *slog.Logger) (*ManagedServer, error) {
	if cfg.Command == "" {
		return nil, fmt.Errorf("command is required for server %s", name)
	}

	return &ManagedServer{
		name:   name,
		cfg:    cfg,
		env:    env,
		logger: logger.With("server", name),
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
		"command", s.cfg.Command,
		"args", s.cfg.Args,
	)

	// Create stdio transport
	s.transport = transport.NewStdio(s.cfg.Command, s.env, s.cfg.Args...)

	// Create client
	s.client = client.NewClient(s.transport)

	// Start the client
	if err := s.client.Start(ctx); err != nil {
		return fmt.Errorf("starting client: %w", err)
	}

	// Initialize the connection
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "Valksor Assern",
		Version: "1.0.0",
	}
	initReq.Params.Capabilities = mcp.ClientCapabilities{}

	_, err := s.client.Initialize(ctx, initReq)
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
