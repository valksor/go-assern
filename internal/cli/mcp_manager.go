// Package cli provides interactive CLI components for assern.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/valksor/go-assern/internal/config"
)

// ScopeType represents whether a server is global or project-specific.
type ScopeType string

const (
	ScopeGlobal  ScopeType = "global"
	ScopeProject ScopeType = "project"
)

// MCPInput collects all user input for adding/editing an MCP server.
type MCPInput struct {
	Name      string
	Scope     ScopeType
	Project   string // if scope is project
	Transport string

	// Stdio fields
	Command string
	Args    []string
	WorkDir string
	Env     map[string]string

	// HTTP/SSE fields
	URL     string
	Headers map[string]string

	// OAuth fields
	OAuth *config.OAuthConfig

	// Common fields
	Allowed   []string
	Disabled  bool
	MergeMode config.MergeMode
}

// MCPManager handles MCP server CRUD operations.
type MCPManager struct {
	globalMCP  *config.MCPConfig
	globalPath string
	localMCP   *config.MCPConfig
	localPath  string
	cwd        string
}

// NewMCPManager creates a manager for MCP operations.
func NewMCPManager() (*MCPManager, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	mgr := &MCPManager{cwd: cwd}

	// Load global config
	globalPath, err := config.GlobalMCPPath()
	if err != nil {
		return nil, fmt.Errorf("getting global MCP path: %w", err)
	}
	mgr.globalPath = globalPath
	mgr.globalMCP, err = config.LoadMCPConfig(globalPath)
	if err != nil {
		return nil, fmt.Errorf("loading global MCP config: %w", err)
	}

	// Load local config if exists
	localDir := config.FindLocalConfigDir(cwd)
	if localDir != "" {
		mgr.localPath = config.LocalMCPPath(localDir)
		if config.FileExists(mgr.localPath) {
			mgr.localMCP, err = config.LoadMCPConfig(mgr.localPath)
			if err != nil {
				return nil, fmt.Errorf("loading local MCP config: %w", err)
			}
		}
	}

	return mgr, nil
}

// NewMCPManagerWithPath creates a manager with a specific working directory.
func NewMCPManagerWithPath(cwd string) (*MCPManager, error) {
	mgr := &MCPManager{cwd: cwd}

	// Load global config
	globalPath, err := config.GlobalMCPPath()
	if err != nil {
		return nil, fmt.Errorf("getting global MCP path: %w", err)
	}
	mgr.globalPath = globalPath
	mgr.globalMCP, err = config.LoadMCPConfig(globalPath)
	if err != nil {
		return nil, fmt.Errorf("loading global MCP config: %w", err)
	}

	// Load local config if exists
	localDir := config.FindLocalConfigDir(cwd)
	if localDir != "" {
		mgr.localPath = config.LocalMCPPath(localDir)
		if config.FileExists(mgr.localPath) {
			mgr.localMCP, err = config.LoadMCPConfig(mgr.localPath)
			if err != nil {
				return nil, fmt.Errorf("loading local MCP config: %w", err)
			}
		}
	}

	return mgr, nil
}

// ServerInfo contains metadata about a server.
type ServerInfo struct {
	Name      string
	Scope     ScopeType
	Transport string
	Server    *config.MCPServer
	Project   string // For project-scoped servers
}

// AddServer adds a new MCP server.
func (m *MCPManager) AddServer(input *MCPInput) error {
	// Check for duplicate name
	if err := m.checkDuplicate(input.Name, ""); err != nil {
		return err
	}

	server := m.inputToMCPServer(input)

	if input.Scope == ScopeGlobal {
		// Add to global
		if m.globalMCP.MCPServers == nil {
			m.globalMCP.MCPServers = make(map[string]*config.MCPServer)
		}
		m.globalMCP.MCPServers[input.Name] = server

		return m.globalMCP.Save(m.globalPath)
	}

	// Add to local
	if m.localMCP == nil {
		// Ensure local directory exists
		localDir := config.FindLocalConfigDir(m.cwd)
		if localDir == "" {
			var err error
			localDir, err = config.EnsureLocalDir(m.cwd)
			if err != nil {
				return fmt.Errorf("creating local config directory: %w", err)
			}
		}
		m.localMCP = config.NewMCPConfig()
		m.localPath = config.LocalMCPPath(localDir)
	}

	if m.localMCP.MCPServers == nil {
		m.localMCP.MCPServers = make(map[string]*config.MCPServer)
	}
	m.localMCP.MCPServers[input.Name] = server

	return m.localMCP.Save(m.localPath)
}

// UpdateServer updates an existing server.
func (m *MCPManager) UpdateServer(name string, input *MCPInput) error {
	// Check for duplicate name if renaming
	if input.Name != name {
		if err := m.checkDuplicate(input.Name, name); err != nil {
			return err
		}
	}

	server := m.inputToMCPServer(input)

	// Find which config contains the server
	if _, ok := m.globalMCP.MCPServers[name]; ok {
		// Delete old name if renaming
		if input.Name != name {
			delete(m.globalMCP.MCPServers, name)
		}
		m.globalMCP.MCPServers[input.Name] = server

		return m.globalMCP.Save(m.globalPath)
	}

	if m.localMCP != nil {
		if _, ok := m.localMCP.MCPServers[name]; ok {
			// Delete old name if renaming
			if input.Name != name {
				delete(m.localMCP.MCPServers, name)
			}
			m.localMCP.MCPServers[input.Name] = server

			return m.localMCP.Save(m.localPath)
		}
	}

	return fmt.Errorf("server %s not found", name)
}

// DeleteServer removes servers.
func (m *MCPManager) DeleteServer(names []string) error {
	modified := false
	var deletedNames []string

	for _, name := range names {
		if _, ok := m.globalMCP.MCPServers[name]; ok {
			delete(m.globalMCP.MCPServers, name)
			deletedNames = append(deletedNames, name)
			modified = true
		}

		if m.localMCP != nil {
			if _, ok := m.localMCP.MCPServers[name]; ok {
				delete(m.localMCP.MCPServers, name)
				// Only add to deletedNames if not already added from global
				found := false
				for _, n := range deletedNames {
					if n == name {
						found = true

						break
					}
				}
				if !found {
					deletedNames = append(deletedNames, name)
				}
				modified = true
			}
		}
	}

	if !modified {
		return errors.New("none of the specified servers were found")
	}

	// Save modified configs
	if err := m.globalMCP.Save(m.globalPath); err != nil {
		return fmt.Errorf("saving global config: %w", err)
	}

	if m.localMCP != nil {
		if err := m.localMCP.Save(m.localPath); err != nil {
			return fmt.Errorf("saving local config: %w", err)
		}
	}

	return nil
}

// ListServers returns all servers with metadata.
func (m *MCPManager) ListServers() []ServerInfo {
	var servers []ServerInfo

	for name, srv := range m.globalMCP.MCPServers {
		servers = append(servers, ServerInfo{
			Name:      name,
			Scope:     ScopeGlobal,
			Transport: detectTransport(srv),
			Server:    srv,
		})
	}

	if m.localMCP != nil {
		for name, srv := range m.localMCP.MCPServers {
			servers = append(servers, ServerInfo{
				Name:      name,
				Scope:     ScopeProject,
				Transport: detectTransport(srv),
				Server:    srv,
			})
		}
	}

	return servers
}

// GetServer retrieves a server by name.
func (m *MCPManager) GetServer(name string) (*config.MCPServer, ScopeType, error) {
	if srv, ok := m.globalMCP.MCPServers[name]; ok {
		return srv, ScopeGlobal, nil
	}

	if m.localMCP != nil {
		if srv, ok := m.localMCP.MCPServers[name]; ok {
			return srv, ScopeProject, nil
		}
	}

	return nil, "", fmt.Errorf("server %s not found", name)
}

// ServerNames returns all server names grouped by scope.
func (m *MCPManager) ServerNames() ([]string, []string) {
	global := make([]string, 0)
	for name := range m.globalMCP.MCPServers {
		global = append(global, name)
	}

	local := make([]string, 0)
	if m.localMCP != nil {
		for name := range m.localMCP.MCPServers {
			local = append(local, name)
		}
	}

	return global, local
}

// inputToMCPServer converts MCPInput to MCPServer.
func (m *MCPManager) inputToMCPServer(input *MCPInput) *config.MCPServer {
	server := &config.MCPServer{
		Command:   input.Command,
		Args:      input.Args,
		Env:       input.Env,
		WorkDir:   input.WorkDir,
		URL:       input.URL,
		Headers:   input.Headers,
		OAuth:     input.OAuth,
		Transport: input.Transport,
	}

	return server
}

// checkDuplicate checks if a server name already exists (excluding the given skipName).
func (m *MCPManager) checkDuplicate(name, skipName string) error {
	if name == skipName {
		return nil
	}

	if _, ok := m.globalMCP.MCPServers[name]; ok {
		return fmt.Errorf("server '%s' already exists in global config", name)
	}

	if m.localMCP != nil {
		if _, ok := m.localMCP.MCPServers[name]; ok {
			return fmt.Errorf("server '%s' already exists in project config", name)
		}
	}

	return nil
}

// detectTransport auto-detects transport type from server config.
func detectTransport(srv *config.MCPServer) string {
	if srv.Transport != "" {
		return srv.Transport
	}

	if srv.Command != "" {
		return "stdio"
	}

	if srv.OAuth != nil {
		return "oauth-http"
	}

	if srv.URL != "" {
		return "http"
	}

	return "unknown"
}
