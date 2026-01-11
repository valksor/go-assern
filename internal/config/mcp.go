// Package config provides configuration types and loading for Assern.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// MCPConfig represents the standard MCP JSON configuration format.
// This matches the format used by Claude Desktop and other MCP clients.
type MCPConfig struct {
	MCPServers map[string]*MCPServer `json:"mcpServers"`
}

// MCPServer represents a single MCP server in the standard format.
type MCPServer struct {
	// Stdio transport fields
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	WorkDir string            `json:"workDir,omitempty"` // Working directory for stdio servers

	// HTTP/SSE transport fields
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"` // Custom HTTP headers (API keys, Bearer tokens)

	// OAuth configuration for authenticated HTTP/SSE transports
	OAuth *OAuthConfig `json:"oauth,omitempty"`

	// Transport type hint: "stdio", "sse", "http", "oauth-sse", "oauth-http" (auto-detected if not specified)
	Transport string `json:"transport,omitempty"`
}

// NewMCPConfig creates a new empty MCPConfig.
func NewMCPConfig() *MCPConfig {
	return &MCPConfig{
		MCPServers: make(map[string]*MCPServer),
	}
}

// LoadMCPConfig reads an MCP configuration from a JSON file.
func LoadMCPConfig(path string) (*MCPConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewMCPConfig(), nil
		}

		return nil, fmt.Errorf("reading mcp config: %w", err)
	}

	return ParseMCPConfig(data)
}

// ParseMCPConfig parses MCP JSON configuration data.
func ParseMCPConfig(data []byte) (*MCPConfig, error) {
	cfg := NewMCPConfig()

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing mcp config: %w", err)
	}

	return cfg, nil
}

// Save writes the MCP configuration to the given path as JSON.
func (c *MCPConfig) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling mcp config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing mcp config: %w", err)
	}

	return nil
}

// ToServerConfigs converts MCPConfig to internal ServerConfig map.
func (c *MCPConfig) ToServerConfigs() map[string]*ServerConfig {
	servers := make(map[string]*ServerConfig, len(c.MCPServers))

	for name, srv := range c.MCPServers {
		servers[name] = &ServerConfig{
			Command:   srv.Command,
			Args:      srv.Args,
			Env:       srv.Env,
			WorkDir:   srv.WorkDir,
			URL:       srv.URL,
			Headers:   srv.Headers,
			OAuth:     srv.OAuth.Clone(),
			Transport: srv.Transport,
			MergeMode: MergeModeOverlay, // Default merge mode
		}
	}

	return servers
}

// Clone creates a deep copy of the MCP configuration.
func (c *MCPConfig) Clone() *MCPConfig {
	if c == nil {
		return nil
	}

	clone := NewMCPConfig()

	for name, srv := range c.MCPServers {
		clone.MCPServers[name] = srv.Clone()
	}

	return clone
}

// Clone creates a deep copy of the MCP server.
func (s *MCPServer) Clone() *MCPServer {
	if s == nil {
		return nil
	}

	clone := &MCPServer{
		Command:   s.Command,
		Args:      make([]string, len(s.Args)),
		Env:       make(map[string]string, len(s.Env)),
		WorkDir:   s.WorkDir,
		URL:       s.URL,
		Headers:   make(map[string]string, len(s.Headers)),
		OAuth:     s.OAuth.Clone(),
		Transport: s.Transport,
	}

	copy(clone.Args, s.Args)

	for k, v := range s.Env {
		clone.Env[k] = v
	}

	for k, v := range s.Headers {
		clone.Headers[k] = v
	}

	return clone
}

// Merge combines two MCP configs, with other taking precedence.
func (c *MCPConfig) Merge(other *MCPConfig) *MCPConfig {
	if c == nil {
		return other.Clone()
	}
	if other == nil {
		return c.Clone()
	}

	result := c.Clone()

	for name, srv := range other.MCPServers {
		result.MCPServers[name] = srv.Clone()
	}

	return result
}
