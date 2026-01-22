// Package cli provides interactive CLI components for assern.
package cli

import (
	"testing"

	"github.com/valksor/go-assern/internal/config"
)

func TestFormatServerList(t *testing.T) {
	tests := []struct {
		name     string
		servers  []ServerInfo
		verbose  bool
		contains []string
	}{
		{
			name:    "empty list",
			servers: []ServerInfo{},
			verbose: false,
			contains: []string{
				"No MCP servers configured",
			},
		},
		{
			name: "global servers only",
			servers: []ServerInfo{
				{
					Name:      "github",
					Scope:     ScopeGlobal,
					Transport: "stdio",
					Server: &config.MCPServer{
						Command: "npx",
						Args:    []string{"-y", "@modelcontextprotocol/server-github"},
					},
				},
				{
					Name:      "filesystem",
					Scope:     ScopeGlobal,
					Transport: "stdio",
					Server: &config.MCPServer{
						Command: "npx",
						Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
					},
				},
			},
			verbose: false,
			contains: []string{
				"Global Servers",
				"github",
				"filesystem",
				"stdio",
				"enabled",
			},
		},
		{
			name: "mixed global and project servers",
			servers: []ServerInfo{
				{
					Name:      "github",
					Scope:     ScopeGlobal,
					Transport: "stdio",
					Server:    &config.MCPServer{Command: "npx"},
				},
				{
					Name:      "jira",
					Scope:     ScopeProject,
					Project:   "work",
					Transport: "http",
					Server:    &config.MCPServer{URL: "https://jira.example.com/mcp"},
				},
			},
			verbose: false,
			contains: []string{
				"Global Servers",
				"github",
				"Project: work",
				"jira",
			},
		},
		{
			name: "verbose output",
			servers: []ServerInfo{
				{
					Name:      "github",
					Scope:     ScopeGlobal,
					Transport: "stdio",
					Server: &config.MCPServer{
						Command: "npx",
						Args:    []string{"-y", "server"},
					},
				},
			},
			verbose: true,
			contains: []string{
				"npx",
				"-y",
				"server",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatServerList(tt.servers, tt.verbose)

			for _, expected := range tt.contains {
				if !containsString(result, expected) {
					t.Errorf("FormatServerList() output does not contain %q\nOutput:\n%s", expected, result)
				}
			}
		})
	}
}

func TestFormatServerDetail(t *testing.T) {
	tests := []struct {
		name     string
		server   *ServerInfo
		contains []string
	}{
		{
			name: "stdio server",
			server: &ServerInfo{
				Name:      "github",
				Scope:     ScopeGlobal,
				Transport: "stdio",
				Server: &config.MCPServer{
					Command: "npx",
					Args:    []string{"-y", "@modelcontextprotocol/server-github"},
					Env:     map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
					WorkDir: "/tmp",
				},
			},
			contains: []string{
				"Name: github",
				"Scope: global",
				"Transport: stdio",
				"Command: npx",
				"Args: -y @modelcontextprotocol/server-github",
				"Working Directory: /tmp",
				"Environment:",
				"GITHUB_TOKEN: ${GITHUB_TOKEN}",
			},
		},
		{
			name: "http server",
			server: &ServerInfo{
				Name:      "api",
				Scope:     ScopeGlobal,
				Transport: "http",
				Server: &config.MCPServer{
					URL: "https://api.example.com/mcp",
					Headers: map[string]string{
						"Authorization": "Bearer ${API_TOKEN}",
					},
				},
			},
			contains: []string{
				"Name: api",
				"Transport: http",
				"URL: https://api.example.com/mcp",
				"Headers:",
				"Authorization: Bearer ${API_TOKEN}",
			},
		},
		{
			name: "oauth server",
			server: &ServerInfo{
				Name:      "enterprise",
				Scope:     ScopeProject,
				Project:   "work",
				Transport: "oauth-http",
				Server: &config.MCPServer{
					URL: "https://enterprise.com/mcp",
					OAuth: &config.OAuthConfig{
						ClientID:              "client-123",
						ClientSecret:          "secret-456",
						RedirectURI:           "http://localhost:8080/callback",
						Scopes:                []string{"read", "write"},
						AuthServerMetadataURL: "https://auth.enterprise.com/.well-known/oauth-authorization-server",
						PKCEEnabled:           true,
					},
				},
			},
			contains: []string{
				"Name: enterprise",
				"Scope: project (work)",
				"Transport: oauth-http",
				"URL: https://enterprise.com/mcp",
				"OAuth:",
				"Client ID: client-123",
				"Client Secret: ***",
				"Scopes: read, write",
				"PKCE: enabled",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatServerDetail(tt.server)

			for _, expected := range tt.contains {
				if !containsString(result, expected) {
					t.Errorf("FormatServerDetail() output does not contain %q\nOutput:\n%s", expected, result)
				}
			}
		})
	}
}

func TestFormatDiff(t *testing.T) {
	oldServer := &ServerInfo{
		Name:      "github",
		Scope:     ScopeGlobal,
		Transport: "stdio",
		Server: &config.MCPServer{
			Command: "node",
			Args:    []string{"old-server.js"},
		},
	}

	newServer := &ServerInfo{
		Name:      "github-v2",
		Scope:     ScopeGlobal,
		Transport: "http",
		Server: &config.MCPServer{
			URL: "https://github.com/mcp",
		},
	}

	result := FormatDiff("github", "github-v2", oldServer, newServer)

	expectedContains := []string{
		"Changes to server 'github'",
		"Name: github -> github-v2",
		"Transport: stdio -> http",
	}

	for _, expected := range expectedContains {
		if !containsString(result, expected) {
			t.Errorf("FormatDiff() output does not contain %q\nOutput:\n%s", expected, result)
		}
	}
}

func TestFormatServer(t *testing.T) {
	// Test the internal formatServer function indirectly through FormatServerList
	servers := []ServerInfo{
		{
			Name:      "test",
			Scope:     ScopeGlobal,
			Transport: "stdio",
			Server: &config.MCPServer{
				Command: "npx",
				Args:    []string{"server.js"},
			},
		},
	}

	result := FormatServerList(servers, false)

	if !containsString(result, "test") {
		t.Error("formatServer() output does not contain server name")
	}

	if !containsString(result, "stdio") {
		t.Error("formatServer() output does not contain transport type")
	}

	if !containsString(result, "enabled") {
		t.Error("formatServer() output does not contain status")
	}
}

func TestGetGlobalPath(t *testing.T) {
	path, err := getGlobalPath()
	if err != nil {
		t.Errorf("getGlobalPath() error = %v", err)
	}

	if path == "" {
		t.Error("getGlobalPath() returned empty string")
	}

	// Should contain ~/.valksor/assern/mcp.json or similar
	if !containsString(path, "mcp.json") {
		t.Errorf("getGlobalPath() = %v, should contain mcp.json", path)
	}
}
